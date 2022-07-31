package config

import (
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"github.com/spf13/viper"
)

const (
	defaultKubeConfigPath = "$HOME/.kube/config"
	defaultListenAddr     = "0.0.0.0:8000"
)

type ConfigFile struct {
	Proxy        Proxy                   `mapstructure:"proxy"`
	Targets      map[string]*Target      `mapstructure:"targets"`
	ProxyTargets map[string]*ProxyTarget `mapstructure:"upstream_proxies"`
}

type Proxy struct {
	ListenAddr string    `mapstructure:"listen_addr"`
	SSL        ServerSSL `mapstructure:"ssl"`
	ACL        ACL       `mapstructure:"target_acl"`
}

func LoadConfig(filepath string) (ConfigFile, error) {
	var config ConfigFile
	if filepath != "" {
		viper.SetConfigFile(filepath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$XDG_CONFIG_HOME/rds-auth-proxy")
		viper.AddConfigPath("$HOME/.config/rds-auth-proxy")
	}
	if err := viper.ReadInConfig(); err != nil {
		return config, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return config, err
	}
	config.Init()
	return config, nil
}

// Init sets up defaults for the config file
func (c *ConfigFile) Init() {
	if c.Targets == nil {
		c.Targets = map[string]*Target{}
	}

	if c.ProxyTargets == nil {
		c.ProxyTargets = map[string]*ProxyTarget{}
	}

	if c.Proxy.ListenAddr == "" {
		c.Proxy.ListenAddr = defaultListenAddr
	}

	c.Proxy.ACL.Init()
	// Set up SSL defaults for all targets if not set
	for key, target := range c.Targets {
		target.Name = key
		// if no SSL keys
		if target.SSL.Mode == "" {
			target.SSL.Mode = pg.SSLRequired
		}
		if target.SSL.Mode != pg.SSLDisabled && target.SSL.ClientCertificatePath == nil {
			target.SSL.ClientCertificatePath = c.Proxy.SSL.ClientCertificatePath
			target.SSL.ClientPrivateKeyPath = c.Proxy.SSL.ClientPrivateKeyPath
		}
	}

	// Set up SSL defaults for all proxies if not set
	for key, target := range c.ProxyTargets {
		target.Name = key
		if target.PortForward != nil && target.PortForward.KubeConfigFilePath == "" {
			target.PortForward.KubeConfigFilePath = defaultKubeConfigPath
			if target.SSL.Mode == "" {
				target.SSL.Mode = pg.SSLDisabled
			}
		}

		if target.SSL.Mode == "" {
			target.SSL.Mode = pg.SSLRequired
		}
		// if no SSL keys
		if target.SSL.Mode != pg.SSLDisabled && target.SSL.ClientCertificatePath == nil {
			target.SSL.ClientCertificatePath = c.Proxy.SSL.ClientCertificatePath
			target.SSL.ClientPrivateKeyPath = c.Proxy.SSL.ClientPrivateKeyPath
		}
	}
}
