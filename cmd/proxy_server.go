package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
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
		redshiftClient, err := aws.NewRedshiftClient(ctx)
		if err != nil {
			return err
		}
		cfg, err := config.LoadConfig(ctx, rdsClient, redshiftClient, filepath)
		if err != nil {
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
				hostConfig, ok := cfg.HostMap[creds.Host]
				if !ok {
					logger.Warn("client attempted to login to unknown host", zap.String("host", creds.Host))
					return fmt.Errorf("host not allowed by ACL, or not configured for this proxy")
				}
				return overrideSSLConfig(creds, hostConfig.SSL)
			})})...,
		)
		if err != nil {
			return err
		}
		config.RefreshTargets(ctx, &cfg, rdsClient, redshiftClient, 1*time.Minute)
		err = manager.Start(ctx)
		return err
	},
}

func init() {
	proxyServerCommand.PersistentFlags().String("configfile", "", "Filepath for proxy config file")
	rootCmd.AddCommand(proxyServerCommand)
}
