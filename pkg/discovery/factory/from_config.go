package discovery

import (
	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	"github.com/mothership/rds-auth-proxy/pkg/discovery/combined"
	"github.com/mothership/rds-auth-proxy/pkg/discovery/rds"
	"github.com/mothership/rds-auth-proxy/pkg/discovery/static"
)

// FromConfig returns a new DiscoveryClient from the settings in your configfile.
func FromConfig(rdsClient aws.RDSClient, c *config.ConfigFile) discovery.Client {
	var staticTargets = make(map[string]config.Target, len(c.Targets))
	for _, target := range c.Targets {
		staticTargets[target.Host] = *target
	}
	return combined.NewCombinedDiscoveryClient([]discovery.Client{
		static.NewStaticDiscoveryClient(staticTargets),
		rds.NewRdsDiscoveryClient(rdsClient, c),
	})
}
