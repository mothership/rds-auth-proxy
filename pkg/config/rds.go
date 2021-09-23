package config

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"go.uber.org/zap"
)

const (
	defaultDatabaseTag = "rds-auth-proxy:db-name"
	localPortTag       = "rds-auth-proxy:local-port"
)

// RefreshRDSTargets searches AWS for allowed dbs updates the target list
func RefreshRDSTargets(ctx context.Context, cfg *ConfigFile, rdsClient aws.RDSClient) (err error) {
	// XXX: Must consume ALL of these, else I think we leak the channel
	resChan := rdsClient.GetPostgresInstances(ctx)
	rdsTargets := map[string]*Target{}
	for result := range resChan {
		if result.Error != nil {
			err = result.Error
			continue
		}
		d := result.Instance
		if d.Endpoint == nil {
			log.Warn("db instance missing endpoint, skipping", zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		if tmpErr := cfg.Proxy.ACL.IsAllowed(d.TagList); tmpErr != nil {
			log.Debug("db instance not allowed by acl", zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		region, err := rdsClient.RegionForInstance(d)
		if err != nil {
			log.Error("failed to detect db region, skipping", zap.Error(err), zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		if !d.IAMDatabaseAuthenticationEnabled {
			log.Warn("db instance does not have IAM auth enabled, skipping", zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		target := &Target{
			Name:            *d.DBInstanceIdentifier,
			Host:            fmt.Sprintf("%+v:%+v", *d.Endpoint.Address, strconv.FormatInt(int64(d.Endpoint.Port), 10)),
			DefaultDatabase: d.DBName,
			SSL: SSL{
				Mode:                  pg.SSLVerifyFull,
				ClientCertificatePath: cfg.Proxy.SSL.ClientCertificatePath,
				ClientPrivateKeyPath:  cfg.Proxy.SSL.ClientPrivateKeyPath,
			},
			Region: region,
		}
		for _, tag := range d.TagList {
			if *tag.Key == defaultDatabaseTag {
				target.DefaultDatabase = tag.Value
			} else if *tag.Key == localPortTag {
				target.LocalPort = tag.Value
			}
		}
		rdsTargets[target.Name] = target
	}
	cfg.RDSTargets = rdsTargets
	cfg.RefreshHostMap()
	return err
}
