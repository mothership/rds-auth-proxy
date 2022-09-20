package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"

	"github.com/AlecAivazis/survey/v2"
	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	discoveryFactory "github.com/mothership/rds-auth-proxy/pkg/discovery/factory"
	"github.com/mothership/rds-auth-proxy/pkg/kubernetes"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/proxy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var proxyClientCommand = &cobra.Command{
	Use:   "client",
	Short: "Launches the localhost proxy",
	Long:  `Runs a localhost proxy service in-cluster for connecting to RDS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logCfg := zap.NewDevelopmentConfig()
		logCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		logCfg.Development = false
		logger, err := logCfg.Build(zap.WithCaller(false))
		if err != nil {
			return err
		}
		log.SetLogger(logger)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		rdsClient, err := aws.NewRDSClient(ctx)
		if err != nil {
			return err
		}
		filepath, err := cmd.Flags().GetString("configfile")
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(filepath)
		if err != nil {
			return err
		}
		discoveryClient := discoveryFactory.FromConfig(rdsClient, &cfg)
		if err := discoveryClient.Refresh(ctx); err != nil {
			return err
		}

		// Look up the proxy target
		proxyTarget, err := getProxyTarget(cmd, cfg.ProxyTargets)
		if err != nil {
			return err
		}

		// Look up the real target name in the target list
		target, err := getTarget(cmd, discoveryClient)
		if err != nil {
			return err
		}
		// Override local port if needed
		if target.LocalPort != nil {
			addr, err := net.ResolveTCPAddr("tcp", cfg.Proxy.ListenAddr)
			if err != nil {
				return err
			}
			cfg.Proxy.ListenAddr = fmt.Sprintf("%s:%s", addr.IP, *target.LocalPort)
		}

		// Optionally grab the password
		pass, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}

		err = printConnectionString(cfg.Proxy.ListenAddr, target)
		if err != nil {
			return err
		}

		if proxyTarget.PortForward != nil {
			// setup port-forward
			prtCmd, err := kubernetes.BuildPortForwardCommand(ctx, proxyTarget.PortForward.KubeConfigFilePath, kubernetes.PortForwardOptions{
				Namespace:  proxyTarget.PortForward.Namespace,
				Deployment: proxyTarget.PortForward.DeploymentName,
				Ports:      []string{fmt.Sprintf("%s:%s", proxyTarget.PortForward.GetLocalPort(), proxyTarget.PortForward.RemotePort)},
				Context:    proxyTarget.PortForward.Context,
			})
			if err != nil {
				return err
			}

			go func() {
				if err := kubernetes.ForwardPort(ctx, prtCmd); err != nil {
					// TODO: blow this up gracefully
					log.Error("k8s port-forward caught error", zap.Error(err), zap.String("listen_addr", proxyTarget.GetHost()))
					panic(err)
				}
				log.Info("k8s port-forward exited", zap.String("listen_addr", proxyTarget.GetHost()))
			}()
			<-prtCmd.ReadyChannel
			ports, err := prtCmd.PortForwarder.GetPorts()
			if err != nil {
				return err
			}
			portUsed := fmt.Sprintf("%d", ports[0].Local)
			proxyTarget.PortForward.LocalPort = &portUsed
			log.Info("started k8s port-forward", zap.String("listen_addr", proxyTarget.GetHost()))
		}

		log.Info("starting client proxy", zap.String("listen_addr", cfg.Proxy.ListenAddr))
		opts, err := proxySSLOptions(cfg.Proxy.SSL)
		if err != nil {
			return err
		}
		var outboundHost string

		if proxyTarget.AwsAuthOnly {
			outboundHost = target.Host
		} else {
			outboundHost = proxyTarget.GetHost()
		}

		manager, err := proxy.NewManager(proxy.MergeOptions(opts, []proxy.Option{
			proxy.WithListenAddress(cfg.Proxy.ListenAddr),
			proxy.WithMode(proxy.ClientSide),
			proxy.WithAWSAuthOnly(proxyTarget.AwsAuthOnly),
			proxy.WithCredentialInterceptor(func(creds *proxy.Credentials) error {
				// Send this connection to the proxy host
				creds.Host =  outboundHost
				// Use provided password, or generate an RDS password to forward through
				if pass != "" {
					creds.Password = pass
				} else if target.IsRDS {
					authToken, err := rdsClient.NewAuthToken(ctx, target.Host, target.Region, creds.Username)
					if err != nil {
						return err
					}
					creds.Password = authToken
				}

				if !proxyTarget.AwsAuthOnly {
					creds.Options["host"] = target.Host
				}

				return overrideSSLConfig(creds, proxyTarget.SSL)
			})})...,
		)
		if err != nil {
			return err
		}

		// Shutdown app on SIGINT/SIGTERM
		signals := make(chan os.Signal, 1)
		go func() {
			_ = manager.Start(ctx)
			close(signals)
		}()
		signal.Notify(signals, os.Interrupt)
		<-signals
		cancel()
		return nil
	},
}

func printConnectionString(listenAddr string, target config.Target) error {
	addr, err := net.ResolveTCPAddr("tcp", listenAddr)
	if err != nil {
		return err
	}
	start := fmt.Sprintf("psql -h %s -p %d", addr.IP, addr.Port)
	if target.DefaultDatabase != nil && *target.DefaultDatabase != "" {
		start += fmt.Sprintf(" -d %s", *target.DefaultDatabase)
	} else {
		start += " -d {your_database}"
	}
	start += " -U {your user}"
	fmt.Printf("Setting up a tunnel to %s\n\nGive this a second, then in a new shell, connect with:\n\n\t%s\n\n", target.Name, start)
	return nil
}

func getProxyTarget(cmd *cobra.Command, targets map[string]*config.ProxyTarget) (*config.ProxyTarget, error) {
	// Look up the proxy target
	proxyName, err := cmd.Flags().GetString("proxy-target")
	if err != nil {
		return nil, err
	}
	proxyTarget, ok := targets[proxyName]
	if ok {
		return proxyTarget, nil
	}

	opts := make([]string, 0, len(targets))
	for name := range targets {
		opts = append(opts, name)
	}

	prompt := &survey.Select{
		Message: "Select an upstream proxy",
		Options: opts,
	}

	err = survey.AskOne(prompt, &proxyName)
	if err != nil {
		return nil, err
	}
	proxyTarget, ok = targets[proxyName]
	if ok {
		return proxyTarget, nil
	}
	return nil, fmt.Errorf("couldn't find a proxy target")
}

func getTarget(cmd *cobra.Command, discoveryClient discovery.Client) (config.Target, error) {
	targetName, err := cmd.Flags().GetString("target")
	if err != nil {
		return config.Target{}, err
	}

	if targetName == "" {
		targets := discoveryClient.GetTargets()
		opts := make([]string, 0, len(targets))
		for _, target := range targets {
			opts = append(opts, target.Name)
		}
		prompt := &survey.Select{
			Message: "Select a database",
			Options: opts,
		}

		if err := survey.AskOne(prompt, &targetName); err != nil {
			return config.Target{}, err
		}
	}
	return discoveryClient.LookupTargetByName(targetName)
}

func overrideSSLConfig(creds *proxy.Credentials, ssl config.SSL) error {
	creds.SSLMode = ssl.Mode
	// If the config wants us to use a specific SSL client cert, load it
	if ssl.ClientCertificatePath != nil {
		// TODO: load sooner / cache
		cert, err := tls.LoadX509KeyPair(*ssl.ClientCertificatePath, *ssl.ClientPrivateKeyPath)
		if err != nil {
			return err
		}
		creds.ClientCertificate = &cert
	}

	// If the config wants us to validate the cert chain goes to a specific root cert for the server proxy
	// load it, and set it
	if ssl.RootCertificatePath != nil {
		rootCABytes, err := ioutil.ReadFile(*ssl.RootCertificatePath)
		if err != nil {
			return err
		}
		decoded, _ := pem.Decode(rootCABytes)
		cert, err := x509.ParseCertificate(decoded.Bytes)
		if err != nil {
			return err
		}
		creds.RootCertificate = cert
	}
	return nil
}

func proxySSLOptions(ssl config.ServerSSL) ([]proxy.Option, error) {
	opts := make([]proxy.Option, 0, 2)
	if ssl.Enabled {
		if ssl.CertificatePath == nil && ssl.PrivateKeyPath == nil {
			opts = append(opts, proxy.WithGeneratedServerCertificate())
		} else if ssl.CertificatePath != nil && ssl.PrivateKeyPath != nil {
			opts = append(opts, proxy.WithServerCertificate(*ssl.CertificatePath, *ssl.PrivateKeyPath))
		} else {
			return opts, fmt.Errorf("bad options: when ssl is enabled, either both a certificate and key must be provided, or neither provided")
		}
	}

	if ssl.ClientCertificatePath == nil && ssl.ClientPrivateKeyPath == nil {
		opts = append(opts, proxy.WithGeneratedClientCertificate())
	} else if ssl.ClientCertificatePath != nil && ssl.ClientPrivateKeyPath != nil {
		opts = append(opts, proxy.WithClientCertificate(*ssl.ClientCertificatePath, *ssl.ClientPrivateKeyPath))
	} else {
		return opts, fmt.Errorf("bad options: either both a client certificate and key must be provided, or neither provided")
	}
	return opts, nil

}

func init() {
	proxyClientCommand.PersistentFlags().String("proxy-target", "default", "Name of the proxy target in the configfile")
	proxyClientCommand.PersistentFlags().String("target", "", "Name of the target, or db instance identifier that you wish to connect to")
	proxyClientCommand.PersistentFlags().String("configfile", "", "Path to the proxy config file")
	_ = proxyClientCommand.MarkPersistentFlagDirname("configfile")
	proxyClientCommand.PersistentFlags().String("password", "", "Password for the user if IAM auth is not set up")
	rootCmd.AddCommand(proxyClientCommand)
}
