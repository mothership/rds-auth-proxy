package discovery

import (
	"context"

	"github.com/mothership/rds-auth-proxy/pkg/config"
)

// Client is for discovering new database servers
type Client interface {
	LookupTargetByHost(host string) (config.Target, error)
	LookupTargetByName(name string) (config.Target, error)
	GetTargets() []config.Target
	Refresh(ctx context.Context) error
}
