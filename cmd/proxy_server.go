package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	discoveryFactory "github.com/mothership/rds-auth-proxy/pkg/discovery/factory"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/proxy"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var proxyServerCommand = &cobra.Command{
	Use:   "server",
	Short: "Launches the server proxy",
	Long:  `Runs a proxy service in-cluster for connecting to RDS.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: make this gracefully shutdown on sigterm / sigint
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		logger := log.NewLogger()
		filepath, err := cmd.Flags().GetString("configfile")
		if err != nil {
			return err
		}
		rdsClient, err := aws.NewRDSClient(ctx)
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

		opts, err := proxySSLOptions(cfg.Proxy.SSL)
		if err != nil {
			return err
		}
		logger.Info("starting server", zap.String("listen_addr", cfg.Proxy.ListenAddr))
		manager, err := proxy.NewManager(proxy.MergeOptions(opts, []proxy.Option{
			proxy.WithListenAddress(cfg.Proxy.ListenAddr),
			proxy.WithMode(proxy.ServerSide),
			proxy.WithCredentialInterceptor(func(creds *proxy.Credentials) error {
				hostConfig, err := discoveryClient.LookupTargetByHost(creds.Host)
				if err != nil {
					logger.Warn("client attempted to login to unknown host", zap.String("host", creds.Host))
					return fmt.Errorf("host not allowed by ACL, or not configured for this proxy")
				}
				return overrideSSLConfig(creds, hostConfig.SSL)
			})})...,
		)
		if err != nil {
			return err
		}
		// TODO: periodic refresh of discovery client
		RefreshTargets(ctx, discoveryClient, 1*time.Minute)
		err = manager.Start(ctx)
		return err
	},
}

func RefreshTargets(ctx context.Context, client discovery.Client, period time.Duration) {
	go func() {
		t := time.NewTicker(period)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				log.Info("starting target refresh", zap.Strings("targets", targetNames(client.GetTargets())))
				if err := client.Refresh(ctx); err != nil {
					log.Warn("refresh failed", zap.Error(err), zap.Strings("targets", targetNames(client.GetTargets())))
				} else {
					log.Info("refresh done", zap.Strings("targets", targetNames(client.GetTargets())))
				}
			}
		}
	}()
}

func targetNames(targets []config.Target) []string {
	instances := make([]string, 0, len(targets))
	for _, target := range targets {
		instances = append(instances, target.Name)
	}
	return instances
}

func init() {
	proxyServerCommand.PersistentFlags().String("configfile", "", "Filepath for proxy config file")
	rootCmd.AddCommand(proxyServerCommand)
}
