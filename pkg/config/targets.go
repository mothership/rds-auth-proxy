package config

import "fmt"

// ProxyTarget is a config block specifying an upstream proxy
type ProxyTarget struct {
	Name string
	Host string `mapstructure:"host"`
	SSL  SSL    `mapstructure:"ssl"`
	// For tunneling the connection through a kubernetes port-forward, only useful
	// for client-side proxy targets
	PortForward *PortForward `mapstructure:"port_forward,omitempty"`
	AwsAuthOnly bool `mapstructure:"aws_auth_only", default:false`
}

// Target is the actual DB server we're connecting to
type Target struct {
	Host string `mapstructure:"host"`
	SSL  SSL    `mapstructure:"ssl"`
	// Hint for showing the default database in the connection string
	DefaultDatabase *string `mapstructure:"database,omitempty"`
	// LocalPort to use instead of the proxy's default ListenAddr port
	LocalPort *string `mapstructure:"local_port,omitempty"`
	// Name in target list, or RDS db instance identifier
	Name string
	// Only set for RDS instances
	Region string
	// Only set for RDS instances
	IsRDS bool
}

// GetHost returns the correct host + port combo for the proxy target
// if the target is port-forwarded, this is a localhost address
// otherwise, it's exposed over a VPN or by some other means.
func (p *ProxyTarget) GetHost() string {
	if p.PortForward == nil {
		return p.Host
	}
	if p.PortForward.LocalPort == nil {
		return "0.0.0.0:0"
	}
	return fmt.Sprintf("0.0.0.0:%s", *p.PortForward.LocalPort)
}

// IsPortForward returns true if this proxy target requires a port-forward connection
func (p *ProxyTarget) IsPortForward() bool {
	return p.PortForward != nil
}
