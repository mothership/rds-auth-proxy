package static

import (
	"context"

	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
)

type StaticDiscoveryClient struct {
	targets map[string]config.Target
}

var _ discovery.Client = (*StaticDiscoveryClient)(nil)

func NewStaticDiscoveryClient(targets map[string]config.Target) *StaticDiscoveryClient {
	return &StaticDiscoveryClient{
		targets: targets,
	}
}

func (s *StaticDiscoveryClient) LookupTargetByHost(host string) (config.Target, error) {
	if target, ok := s.targets[host]; ok {
		return target, nil
	}
	return config.Target{}, discovery.ErrTargetNotFound
}

func (s *StaticDiscoveryClient) LookupTargetByName(name string) (config.Target, error) {
	for _, target := range s.targets {
		if target.Name == name {
			return target, nil
		}
	}
	return config.Target{}, discovery.ErrTargetNotFound
}

func (s *StaticDiscoveryClient) GetTargets() []config.Target {
	targetList := make([]config.Target, 0, len(s.targets))
	for _, target := range s.targets {
		targetList = append(targetList, target)
	}
	return targetList
}

func (s *StaticDiscoveryClient) Refresh(ctx context.Context) error {
	return nil
}
