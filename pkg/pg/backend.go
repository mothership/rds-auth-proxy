package pg

import (
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	pgproto3 "github.com/jackc/pgproto3/v2"
)

const GSSENCecNotAllowed byte = 'N'
const SSLNotAllowed byte = 'N'
const SSLAllowed byte = 'S'

// Backend acts as the postgres front-end client (ex: psql)
type Backend interface {
	io.Closer
	Send(msg pgproto3.BackendMessage) error
	SendRaw([]byte) error
	Receive() (pgproto3.FrontendMessage, error)
	ReceiveRaw() ([]byte, error)
}

// SendOnlyBackend allows only the send operation to be accessed for network safety
type SendOnlyBackend interface {
	Send(msg pgproto3.BackendMessage) error
}

// PostgresBackend implements a postgres backend client
type PostgresBackend struct {
	backend     *pgproto3.Backend
	connection  net.Conn
	IdleTimeout time.Duration
	mutex       sync.Mutex
}

// BackendOption allows us to specify options
type BackendOption func(f *PostgresBackend) error

// NewBackend returns a new postgres backend
func NewBackend(conn net.Conn, opts ...BackendOption) (*PostgresBackend, error) {
	f := &PostgresBackend{
		backend:     pgproto3.NewBackend(pgproto3.NewChunkReader(conn), conn),
		connection:  conn,
		IdleTimeout: readTimeout,
		mutex:       sync.Mutex{},
	}

	for _, opt := range opts {
		err := opt(f)
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Send sends a backend message to the backend
func (b *PostgresBackend) Send(msg pgproto3.BackendMessage) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.backend.Send(msg)
}

// SendRaw sends arbitrary bytes to a backend
func (b *PostgresBackend) SendRaw(msg []byte) error {
	_, err := b.connection.Write(msg)
	return err
}

// Receive accepts a message from the backend, or errors if nothing is read
// within the idle timeout. Returns io.ErrUnexpectedEOF if the connection has
// been closed.
func (b *PostgresBackend) Receive() (pgproto3.FrontendMessage, error) {
	_ = b.connection.SetReadDeadline(time.Now().Add(b.IdleTimeout))
	return b.backend.Receive()
}

// ReceiveRaw accepts a message from the backend, or errors if nothing
// is read within the idle timeout.  Returns io.ErrUnexpectedEOF if the
// connection has been closed.
func (b *PostgresBackend) ReceiveRaw() ([]byte, error) {
	// Postgres send buffers are at least this large
	response := make([]byte, 8192)
	_ = b.connection.SetReadDeadline(time.Now().Add(b.IdleTimeout))
	readBytes, err := b.connection.Read(response)
	if err != nil && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return response[:readBytes], err
}

// Close closes the underlying connection
func (b *PostgresBackend) Close() error {
	return b.connection.Close()
}

// SetupConnection sets up an inbound connection and extracts the login information
// This will always return the existing connection, unless it had to upgrade to an SSL
// connection.
func (b *PostgresBackend) SetupConnection(cert *tls.Certificate) (map[string]string, error) {
	for {
		message, err := b.backend.ReceiveStartupMessage()
		if err != nil {
			return nil, err
		}
		switch msg := message.(type) {
		case *pgproto3.StartupMessage:
			return msg.Parameters, nil
		case *pgproto3.SSLRequest:
			if cert == nil {
				err = b.SendRaw([]byte{SSLNotAllowed})
				if err != nil {
					return nil, err
				}
				continue
			}
			err = b.SendRaw([]byte{SSLAllowed})
			if err != nil {
				return nil, err
			}
			b.connection = UpgradeServer(b.connection, cert)
			b.backend = pgproto3.NewBackend(pgproto3.NewChunkReader(b.connection), b.connection)
			continue
		case *pgproto3.GSSEncRequest:
			// Would need more research to offer GSS enc.
			err = b.SendRaw([]byte{GSSENCecNotAllowed})
			if err != nil {
				return nil, err
			}
			continue
		}
	}
}
