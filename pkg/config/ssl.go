package config

import (
	"github.com/mothership/rds-auth-proxy/pkg/pg"
)

// SSL represents settings for upstream (RDS instances, pg instances)
type SSL struct {
	// Optional client certificate to use
	ClientCertificatePath *string `mapstructure:"client_certificate,omitempty"`
	// Optional client private key to use
	ClientPrivateKeyPath *string `mapstructure:"client_private_key,omitempty"`
	// SSL mode to verify upstream connection, defaults to "verify-full"
	Mode pg.SSLMode `mapstructure:"mode,omitempty"`
	// Path to a root certificate if the certificate is
	// not already in the system roots
	RootCertificatePath *string `mapstructure:"root_certificate"`
}

// ServerSSL is SSL settings for the proxy server
type ServerSSL struct {
	Enabled               bool    `mapstructure:"enabled"`
	CertificatePath       *string `mapstructure:"certificate,omitempty"`
	PrivateKeyPath        *string `mapstructure:"private_key,omitempty"`
	ClientCertificatePath *string `mapstructure:"client_certificate,omitempty"`
	ClientPrivateKeyPath  *string `mapstructure:"client_private_key,omitempty"`
}
