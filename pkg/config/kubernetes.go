package config

// PortForward represents kubernetes port-forward config for tunneling a connection to the server-side proxy
type PortForward struct {
	Namespace      string `mapstructure:"namespace"`
	DeploymentName string `mapstructure:"deployment"`
	RemotePort     string `mapstructure:"remote_port"`
	// Optional, if not set "0" is used
	LocalPort          *string `mapstructure:"local_port"`
	Context            string  `mapstructure:"context"`
	KubeConfigFilePath string  `mapstructure:"kube_config"`
}

// GetLocalPort returns the local port to be used for the port-forward
func (p *PortForward) GetLocalPort() string {
	if p.LocalPort != nil {
		return *p.LocalPort
	}
	return "0"
}
