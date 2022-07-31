package combined

import (
	"context"

	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
)

type CombinedDiscoveryClient struct {
	clients []discovery.Client
}

var _ discovery.Client = (*CombinedDiscoveryClient)(nil)

func NewCombinedDiscoveryClient(clients []discovery.Client) *CombinedDiscoveryClient {
	return &CombinedDiscoveryClient{
		clients: clients,
	}
}

func (c *CombinedDiscoveryClient) LookupTargetByHost(host string) (config.Target, error) {
	for _, client := range c.clients {
		if t, err := client.LookupTargetByHost(host); err == nil {
			return t, nil
		}
	}
	return config.Target{}, discovery.ErrTargetNotFound
}

func (c *CombinedDiscoveryClient) LookupTargetByName(name string) (config.Target, error) {
	for _, client := range c.clients {
		if t, err := client.LookupTargetByName(name); err == nil {
			return t, nil
		}
	}
	return config.Target{}, discovery.ErrTargetNotFound
}

func (c *CombinedDiscoveryClient) GetTargets() []config.Target {
	targetList := make([]config.Target, 0, 16)
	for _, client := range c.clients {
		targetList = append(targetList, client.GetTargets()...)
	}
	return targetList
}

func (c *CombinedDiscoveryClient) Refresh(ctx context.Context) error {
	for _, client := range c.clients {
		if err := client.Refresh(ctx); err != nil {
			return err
		}
	}
	return nil
}
