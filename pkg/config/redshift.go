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
	redshiftDefaultDatabaseTag = "rds-auth-proxy:db-name"
	redshiftLocalPortTag       = "rds-auth-proxy:local-port"
)

// RefreshRedshiftTargets searches AWS for allowed dbs updates the target list
func RefreshRedshiftTargets(ctx context.Context, cfg *ConfigFile, redshiftClient aws.RedshiftClient) (err error) {
	// XXX: Must consume ALL of these, else I think we leak the channel
	resChan := redshiftClient.GetRedshiftInstances(ctx)
	redshiftTargets := map[string]*Target{}
	for result := range resChan {
		if result.Error != nil {
			err = result.Error
			continue
		}
		d := result.Instance
		if d.Endpoint == nil {
			log.Warn("db instance missing endpoint, skipping", zap.String("name", *d.ClusterIdentifier))
			continue
		}

		//TODO: refactor to support redshift ACL filter
		//if tmpErr := cfg.Proxy.ACL.IsAllowed(d.Tags); tmpErr != nil {
		//	log.Debug("redshift not allowed by acl", zap.String("name", *d.ClusterIdentifier))
		//	continue
		//}

		region, err := redshiftClient.RegionForInstance(d)
		if err != nil {
			log.Error("failed to detect redshift region, skipping", zap.Error(err), zap.String("name", *d.ClusterIdentifier))
			continue
		}

		target := &Target{
			Name:            *d.ClusterIdentifier,
			Host:            fmt.Sprintf("%+v:%+v", *d.Endpoint.Address, strconv.FormatInt(int64(d.Endpoint.Port), 10)),
			DefaultDatabase: d.DBName,
			SSL: SSL{
				Mode:                  pg.SSLVerifyFull,
				ClientCertificatePath: cfg.Proxy.SSL.ClientCertificatePath,
				ClientPrivateKeyPath:  cfg.Proxy.SSL.ClientPrivateKeyPath,
			},
			Region: region,
		}

		for _, tag := range d.Tags {
			if *tag.Key == redshiftDefaultDatabaseTag {
				target.DefaultDatabase = tag.Value
			} else if *tag.Key == redshiftLocalPortTag {
				target.LocalPort = tag.Value
			}
		}
		redshiftTargets[target.Name] = target
	}
	cfg.RedshiftTargets = redshiftTargets
	cfg.RefreshHostMap()
	return err
}
