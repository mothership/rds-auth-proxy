package rds

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	"github.com/mothership/rds-auth-proxy/pkg/log"
	"github.com/mothership/rds-auth-proxy/pkg/pg"
	"go.uber.org/zap"
)

const (
	defaultDatabaseTag = "rds-auth-proxy:db-name"
	localPortTag       = "rds-auth-proxy:local-port"
)

type RdsDiscoveryClient struct {
	targetLock *sync.RWMutex
	config     *config.ConfigFile
	client     aws.RDSClient
	rdsTargets map[string]config.Target
}

var _ discovery.Client = (*RdsDiscoveryClient)(nil)

func NewRdsDiscoveryClient(client aws.RDSClient, cfg *config.ConfigFile) *RdsDiscoveryClient {
	return &RdsDiscoveryClient{
		targetLock: &sync.RWMutex{},
		config:     cfg,
		client:     client,
		rdsTargets: map[string]config.Target{},
	}
}

func (r *RdsDiscoveryClient) LookupTargetByHost(host string) (config.Target, error) {
	r.targetLock.RLock()
	defer r.targetLock.RUnlock()
	if target, ok := r.rdsTargets[host]; ok {
		return target, nil
	}
	return config.Target{}, discovery.ErrTargetNotFound

}

func (r *RdsDiscoveryClient) LookupTargetByName(name string) (config.Target, error) {
	r.targetLock.RLock()
	defer r.targetLock.RUnlock()
	for _, target := range r.rdsTargets {
		if target.Name == name {
			return target, nil
		}
	}
	return config.Target{}, discovery.ErrTargetNotFound
}

func (r *RdsDiscoveryClient) GetTargets() []config.Target {
	r.targetLock.RLock()
	defer r.targetLock.RUnlock()
	targetList := make([]config.Target, 0, len(r.rdsTargets))
	for _, target := range r.rdsTargets {
		targetList = append(targetList, target)
	}
	return targetList
}

// RefreshRDSTargets searches AWS for allowed dbs updates the target list
func (r *RdsDiscoveryClient) Refresh(ctx context.Context) (err error) {
	// XXX: Must consume ALL of these, else I think we leak the channel
	resChan := r.client.GetPostgresInstances(ctx)
	rdsTargets := map[string]config.Target{}
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

		if tmpErr := r.config.Proxy.ACL.IsAllowed(d.TagList); tmpErr != nil {
			log.Debug("db instance not allowed by acl", zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		region, regionErr := r.client.RegionForInstance(d)
		if regionErr != nil {
			log.Error("failed to detect db region, skipping", zap.Error(regionErr), zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		if !d.IAMDatabaseAuthenticationEnabled {
			log.Warn("db instance does not have IAM auth enabled, skipping", zap.String("name", *d.DBInstanceIdentifier))
			continue
		}

		target := config.Target{
			Name:            *d.DBInstanceIdentifier,
			Host:            fmt.Sprintf("%+v:%+v", *d.Endpoint.Address, strconv.FormatInt(int64(d.Endpoint.Port), 10)),
			DefaultDatabase: d.DBName,
			SSL: config.SSL{
				Mode:                  pg.SSLVerifyFull,
				ClientCertificatePath: r.config.Proxy.SSL.ClientCertificatePath,
				ClientPrivateKeyPath:  r.config.Proxy.SSL.ClientPrivateKeyPath,
			},
			Region: region,
			IsRDS:  true,
		}
		for _, tag := range d.TagList {
			if *tag.Key == defaultDatabaseTag {
				target.DefaultDatabase = tag.Value
			} else if *tag.Key == localPortTag {
				target.LocalPort = tag.Value
			}
		}
		rdsTargets[target.Host] = target
	}

	if err == nil {
		r.targetLock.Lock()
		defer r.targetLock.Unlock()
		r.rdsTargets = rdsTargets
	}
	return err
}
