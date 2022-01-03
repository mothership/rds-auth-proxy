package config

import (
	"context"
	"time"

	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"go.uber.org/zap"
)

// RefreshTargets refreshes the proxy target list on an interval
func RefreshTargets(ctx context.Context, cfg *ConfigFile, rdsClient aws.RDSClient, redshiftClient aws.RedshiftClient, period time.Duration) {
	go func() {
		t := time.NewTicker(period)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				log.Info("starting target refresh", zap.Strings("targets", targetNames(cfg.RDSTargets)))
				_ = RefreshRDSTargets(ctx, cfg, rdsClient)
				log.Info("refresh done", zap.Strings("targets", targetNames(cfg.RDSTargets)))
				log.Info("starting target refresh", zap.Strings("targets", targetNames(cfg.RedshiftTargets)))
				_ = RefreshRedshiftTargets(ctx, cfg, redshiftClient)
				log.Info("refresh done", zap.Strings("targets", targetNames(cfg.RedshiftTargets)))
			}
		}
	}()
}

func targetNames(targets map[string]*Target) []string {
	instances := make([]string, 0, len(targets))
	for name := range targets {
		instances = append(instances, name)
	}
	return instances
}
