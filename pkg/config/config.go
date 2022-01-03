package config

import (
	"context"
	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"github.com/spf13/viper"
	"strings"
)

const (
	defaultKubeConfigPath = "$HOME/.kube/config"
	defaultListenAddr     = "0.0.0.0:8000"
)

type ConfigFile struct {
	Proxy        	Proxy                   `mapstructure:"proxy"`
	Targets      	map[string]*Target      `mapstructure:"targets"`
	ProxyTargets 	map[string]*ProxyTarget `mapstructure:"upstream_proxies"`
	RDSTargets   	map[string]*Target
	RedshiftTargets map[string]*Target
	HostMap      	map[string]*Target
}

type Proxy struct {
	ListenAddr string    `mapstructure:"listen_addr"`
	SSL        ServerSSL `mapstructure:"ssl"`
	ACL        ACL       `mapstructure:"target_acl"`
}

func LoadConfig(ctx context.Context, rdsClient aws.RDSClient, redshiftClient aws.RedshiftClient ,filepath string) (ConfigFile, error) {
	cfg, err := loadConfig(filepath)
	if err != nil {
		return cfg, err
	}

	err = RefreshRedshiftTargets(ctx, &cfg, redshiftClient)
	if err != nil {
		if !strings.Contains(err.Error(), "StatusCode: 403") {
			return cfg, err
		}
		log.Warn("User Missing Redshift Permissions")
	}

	err = RefreshRDSTargets(ctx, &cfg, rdsClient)
	if err != nil {
		if !strings.Contains(err.Error(), "StatusCode: 403") {
			return cfg, err
		}
		log.Warn("User Missing RDS Permissions")
	}
	return cfg, nil
}

func loadConfig(filepath string) (config ConfigFile, err error) {
	if filepath != "" {
		viper.SetConfigFile(filepath)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$XDG_CONFIG_HOME/rds-auth-proxy")
		viper.AddConfigPath("$HOME/.config/rds-auth-proxy")
	}
	err = viper.ReadInConfig()
	if err != nil {
		return
	}
	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}
	config.Init()
	return
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

	// Copy any manually specified targets into the hostmap for easier lookup
	c.HostMap = make(map[string]*Target, len(c.Targets))
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
		c.HostMap[target.Host] = target
	}

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

// RefreshHostMap updates the list of hosts the proxy knows about
func (c *ConfigFile) RefreshHostMap() {
	hostMap := make(map[string]*Target, len(c.Targets)+len(c.RDSTargets))
	for _, target := range c.RDSTargets {
		hostMap[target.Host] = target
	}

	for _, target := range c.Targets {
		hostMap[target.Host] = target
	}
	c.HostMap = hostMap
}
