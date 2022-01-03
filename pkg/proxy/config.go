package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	pgproto3 "github.com/jackc/pgproto3/v2"
	"github.com/mothership/rds-auth-proxy/pkg/cert"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
)

// Credentials represents connection details to an upstream database or proxy
type Credentials struct {
	Host     string
	Database string
	Username string
	Password string
	// Misc connection parameters to be passed along
	Options map[string]string
	// SSL Settings for the outbound connection
	SSLMode           pg.SSLMode
	ClientCertificate *tls.Certificate
	RootCertificate   *x509.Certificate
}

// CredentialInterceptor provides a way to update credentials being forwarded
// to the server proxy
type CredentialInterceptor func(creds *Credentials) error

// Mode indicates what kind of mode the proxy is in
type Mode int

const (
	// ClientSide proxy mode is for running on the end-user laptop
	ClientSide Mode = iota
	// ServerSide proxy mode is for the in-cluster
	ServerSide
)

// Config contains the various options for setting up the proxy
type Config struct {
	ServerCertificate        *tls.Certificate
	DefaultClientCertificate *tls.Certificate
	ListenAddress            *net.TCPAddr
	CredentialInterceptor    CredentialInterceptor
	QueryInterceptor         QueryInterceptor
	Mode                     Mode
	AwsAuthOnly				 bool `default:false`
}

// QueryInterceptor provides a way to define custom behavior for handling messages
type QueryInterceptor func(frontend pg.SendOnlyFrontend, backend pg.SendOnlyBackend, msg *pgproto3.Query) error

// WillSendManually lets the proxy know that QueryInterceptor will handle sending the message
var WillSendManually = fmt.Errorf("sending manually")

// Option lets you set a config option
type Option func(*Config) error

// WithMode sets the mode of the proxy
func WithMode(mode Mode) Option {
	return func(c *Config) error {
		if mode != ServerSide && mode != ClientSide {
			return fmt.Errorf("invalid mode: %d", mode)
		}
		c.Mode = mode
		return nil
	}
}

// WithListenAddress sets the IP/port that the proxy will accept connections on
func WithListenAddress(addr string) Option {
	return func(c *Config) error {
		listenAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return err
		}

		c.ListenAddress = listenAddr
		return nil
	}
}

func WithAWSAuthOnly(aws_auth_only bool) Option {
	return func(c *Config) error {
		c.AwsAuthOnly = aws_auth_only
		return nil
	}
}

// WithCredentialInterceptor sets the credential retrieval strategy
func WithCredentialInterceptor(credFactory CredentialInterceptor) Option {
	return func(c *Config) error {
		c.CredentialInterceptor = credFactory
		return nil
	}
}

// WithServerCertificate sets the SSL settings for the proxy
func WithServerCertificate(certPath, keyPath string) Option {
	return func(c *Config) (err error) {
		if certPath == "" {
			return fmt.Errorf("certificate path not set")
		}
		if keyPath == "" {
			return fmt.Errorf("private key path not set")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return err
		}
		c.ServerCertificate = &cert
		return nil
	}
}

// WithGeneratedServerCertificate generates a self-signed server certificate for the proxy
func WithGeneratedServerCertificate() Option {
	return func(c *Config) (err error) {
		certBytes, keyBytes, err := cert.GenerateSelfSignedCert("localhost,127.0.0.1", false)
		if err != nil {
			return err
		}
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return err
		}
		c.ServerCertificate = &cert
		return nil
	}
}

// WithClientCertificate sets up the default client certificates
func WithClientCertificate(certPath, keyPath string) Option {
	return func(c *Config) (err error) {
		if certPath == "" {
			return fmt.Errorf("client certificate path not set")
		}
		if keyPath == "" {
			return fmt.Errorf("client private key path not set")
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return err
		}
		c.DefaultClientCertificate = &cert
		return nil
	}
}

// WithGeneratedClientCertificate generates the default client certificates
func WithGeneratedClientCertificate() Option {
	return func(c *Config) (err error) {
		certBytes, keyBytes, err := cert.GenerateSelfSignedCert("localhost,127.0.0.1", false)
		if err != nil {
			return err
		}
		cert, err := tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return err
		}
		c.DefaultClientCertificate = &cert
		return nil
	}
}

// WithQueryInterceptor adds a function for custom message handling
func WithQueryInterceptor(interceptor QueryInterceptor) Option {
	return func(c *Config) (err error) {
		c.QueryInterceptor = interceptor
		return nil
	}
}

// MergeOptions is a helper to merge an option list
func MergeOptions(lists ...[]Option) []Option {
	opts := []Option{}
	for _, l := range lists {
		opts = append(opts, l...)
	}
	return opts
}
