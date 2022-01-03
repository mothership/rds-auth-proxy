package proxy

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"

	pgproto3 "github.com/jackc/pgproto3/v2"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"go.uber.org/zap"
)

var connectionID = uint64(0)

// Proxy - Manages a Proxy connection, piping data between proxy and remote.
type Proxy struct {
	ID           uint64
	logger       *zap.Logger
	backend      *pg.PostgresBackend
	frontend     *pg.PostgresFrontend
	waiter       sync.WaitGroup
	errChan      chan errorWrapper
	shutdownChan chan bool
	config       *Config
}

// newProxy returns a new Proxy that will handle a client connection and open
// a downstream connection to the Postgres server
func newProxy(clientConn net.Conn, errChan chan errorWrapper, config *Config) *Proxy {
	// XXX: can't error if no options are passed
	backend, _ := pg.NewBackend(clientConn)
	shutdownChan := make(chan bool, 1)
	connectionID++
	return &Proxy{
		ID:           connectionID,
		shutdownChan: shutdownChan,
		backend:      backend,
		logger:       log.With(zap.Uint64("connectionID", connectionID)),
		errChan:      errChan,
		waiter:       sync.WaitGroup{},
		config:       config,
	}
}

func (p *Proxy) notifyError(err error) error {
	msg := &pgproto3.ErrorResponse{Severity: "FATAL", Message: err.Error()}
	if authErr, ok := err.(*pg.AuthFailedError); ok {
		msg = authErr.ErrMsg
	}
	_ = p.backend.Send(msg)
	p.errChan <- errorWrapper{ConnectionID: p.ID, Error: err}
	return err
}

func (p *Proxy) notifyStopped() {
	p.errChan <- errorWrapper{ConnectionID: p.ID, Error: nil}
}

// Stop shuts the proxy down and cleans up the connections
func (p *Proxy) Stop() {
	close(p.shutdownChan)
	p.waiter.Wait()
}

// Start boots the proxy
func (p *Proxy) Start() error {
	defer p.backend.Close()
	p.logger.Info("starting connection")
	// First, set up the connection with our client (ex: psql)
	// and extract the connection parameters from the startup message
	connectParams, err := p.backend.SetupConnection(p.config.ServerCertificate)
	if err != nil {
		return p.notifyError(err)
	}
	// Get credentials
	creds := p.ParseCredentials(connectParams)
	if err := p.config.CredentialInterceptor(&creds); err != nil {
		return p.notifyError(err)
	}
	// Next, establish a connection with the upstream database
	p.logger.Info("connecting to upstream postgres server", zap.String("postgres_server", creds.Host))
	connection, err := pg.Connect(creds.Host, creds.SSLMode, creds.ClientCertificate, creds.RootCertificate)
	if err != nil {
		return p.notifyError(err)
	}

	// XXX: can't error without options
	frontend, _ := pg.NewFrontend(connection)
	p.frontend = frontend
	defer p.frontend.Close()

	p.logger.Info("connected to upstream postgres server", zap.String("postgres_server", creds.Host))

	p.logger.Debug("sending startup message",
		zap.Bool("aws_auth_only", p.config.AwsAuthOnly),
		zap.String("postgres_server", creds.Host),
		zap.String("user", creds.Username),
		zap.Any("options", creds.Options),
	)

	// If we're in client proxy mode, forward the password in the StartupMessage
	if (p.config.Mode == ClientSide && !p.config.AwsAuthOnly) && creds.Password != "" {
		creds.Options["password"] = creds.Password
	}
	// Now send our own StartupMessage, and pass thru any remaining connection parameters.
	startupMessage := createStartupMessage(creds.Username, creds.Database, creds.Options)
	if err = frontend.SendRaw(startupMessage.Encode(nil)); err != nil {
		return p.notifyError(err)
	}

	// Even if we're in server mode, don't bother intercepting the startup message response
	// UNLESS we have the password/auth credentials to handle it. This lets the user use
	// the proxy normally, for instance, if they are using it without IAM auth
	if (p.config.Mode == ServerSide || p.config.AwsAuthOnly) && creds.Password != "" {
		// Fetch the response to our startup message, most likely this is going to be a request
		// for us to authenticate. Assuming it is, forward the password we collected.
		p.logger.Debug("handling upstream authentication", zap.String("postgres_server", creds.Host))
		err = p.frontend.HandleAuthenticationRequest(creds.Username, creds.Password)
		if err != nil {
			return p.notifyError(err)
		}
		p.logger.Debug("authed successfully with upstream", zap.String("postgres_server", creds.Host))
		// Send the auth result down to the client (ex: psql)
		err = p.backend.Send(&pgproto3.AuthenticationOk{})
		if err != nil {
			return p.notifyError(errors.New("failed to send auth"))
		}
		p.logger.Debug("notified client of auth result", zap.String("postgres_server", creds.Host))
	}

	// Now move to generic TLS/TCP proxy
	p.logger.Info("startup success, starting full proxy", zap.String("postgres_server", creds.Host))
	p.waiter.Add(2)
	go p.proxyToServer()
	go p.proxyToClient()
	// wait for close...
	p.waiter.Wait()
	return nil
}

func (p *Proxy) proxyToServer() {
	idleTimeout := 5 * time.Minute
	maxTimeouts := int64(int64(idleTimeout) / int64(p.backend.IdleTimeout))
	timeouts := int64(0)
	defer p.waiter.Done()
	for {
		select {
		case <-p.shutdownChan:
			return
		default:
			msg, err := p.backend.Receive()
			if err != nil {
				if isRetryableError(err) {
					timeouts++
					if timeouts < maxTimeouts {
						continue
					}
				}
				_ = p.notifyError(err)
				return
			}
			timeouts = 0

			switch castedMsg := msg.(type) {
			case *pgproto3.Terminate:
				err := p.frontend.Send(castedMsg)
				if err != nil {
					_ = p.notifyError(err)
					return
				}
				p.logger.Debug("got disconnected message")
				p.notifyStopped()
				return
			case *pgproto3.Query:
				p.logger.Debug("got query message from client")
				if p.config.QueryInterceptor != nil {
					if err := p.config.QueryInterceptor(p.frontend, p.backend, castedMsg); err != nil {
						if err != WillSendManually {
							_ = p.notifyError(err)
							return
						}
						continue
					}
				}
				err := p.frontend.Send(castedMsg)
				if err != nil {
					_ = p.notifyError(err)
					return
				}
			default:
				p.logger.Debug("got message from client")
				err := p.frontend.Send(castedMsg)
				if err != nil {
					_ = p.notifyError(err)
					return
				}
			}
		}
	}
}

func (p *Proxy) proxyToClient() {
	idleTimeout := 5 * time.Minute
	maxTimeouts := int64(int64(idleTimeout) / int64(p.frontend.IdleTimeout))
	timeouts := int64(0)
	defer p.waiter.Done()
	for {
		select {
		case <-p.shutdownChan:
			return
		default:
			msg, err := p.frontend.Receive()
			if err != nil {
				if isRetryableError(err) {
					timeouts++
					if timeouts < maxTimeouts {
						continue
					}
				}
				_ = p.notifyError(err)
				return
			}
			timeouts = 0
			p.logger.Debug("got message from server")
			err = p.backend.Send(msg)
			if err != nil {
				_ = p.notifyError(err)
				return
			}
		}
	}
}

func createStartupMessage(username string, database string, options map[string]string) pgproto3.StartupMessage {
	params := map[string]string{
		"user":     username,
		"database": database,
	}
	for key, value := range options {
		params[key] = value
	}

	return pgproto3.StartupMessage{
		ProtocolVersion: pgproto3.ProtocolVersionNumber,
		Parameters:      params,
	}
}

func isRetryableError(err error) bool {
	// These errors are expected in periods of no query activity.
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	return false
}

// ParseCredentials takes connection parameters and turns them into Credentials
func (p *Proxy) ParseCredentials(connectionParams map[string]string) Credentials {
	extracted := []string{"host", "password", "user", "database"}
	creds := Credentials{
		Host:              connectionParams["host"],
		Password:          connectionParams["password"],
		Username:          connectionParams["user"],
		Database:          connectionParams["database"],
		SSLMode:           pg.SSLRequired,
		ClientCertificate: p.config.DefaultClientCertificate,
	}
	for _, key := range extracted {
		delete(connectionParams, key)
	}
	creds.Options = connectionParams
	return creds
}
