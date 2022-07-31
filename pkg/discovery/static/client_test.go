package static_test

import (
	"testing"

	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	. "github.com/mothership/rds-auth-proxy/pkg/discovery/static"
)

func TestStaticDiscoveryClientHostLookupFailures(t *testing.T) {
	staticTargets := map[string]config.Target{
		"test.com:5432": makeTarget("test", "test.com:5432"),
	}
	cases := []struct {
		Targets  map[string]config.Target
		Host     string
		Expected error
	}{
		{
			Targets:  nil,
			Host:     "test.com:5432",
			Expected: discovery.ErrTargetNotFound,
		},
		{
			Targets:  staticTargets,
			Host:     "missing",
			Expected: discovery.ErrTargetNotFound,
		},
	}
	for idx, test := range cases {
		client := NewStaticDiscoveryClient(test.Targets)
		_, err := client.LookupTargetByHost(test.Host)
		if test.Expected != err {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, err)
		}
	}
}

func TestStaticDiscoveryClientHostLookupSuccess(t *testing.T) {
	staticTargets := map[string]config.Target{
		"test.com:5432": makeTarget("test", "test.com:5432"),
	}
	cases := []struct {
		Targets  map[string]config.Target
		Host     string
		Expected config.Target
	}{
		{
			Targets:  staticTargets,
			Host:     "test.com:5432",
			Expected: staticTargets["test.com:5432"],
		},
	}
	for idx, test := range cases {
		client := NewStaticDiscoveryClient(test.Targets)
		target, err := client.LookupTargetByHost(test.Host)
		if err != nil {
			t.Fatalf("[Case %d] unexpected error: %s", idx, err)
		}
		if test.Expected != target {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestStaticDiscoveryClientNameLookupSuccess(t *testing.T) {
	staticTargets := map[string]config.Target{
		"test.com:5432": makeTarget("test", "test.com:5432"),
	}
	cases := []struct {
		Targets  map[string]config.Target
		Name     string
		Expected config.Target
	}{
		{
			Targets:  staticTargets,
			Name:     "test",
			Expected: staticTargets["test.com:5432"],
		},
	}
	for idx, test := range cases {
		client := NewStaticDiscoveryClient(test.Targets)
		target, err := client.LookupTargetByName(test.Name)
		if err != nil {
			t.Fatalf("[Case %d] unexpected error: %s", idx, err)
		}
		if test.Expected != target {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestStaticDiscoveryClientNameLookupFailures(t *testing.T) {
	staticTargets := map[string]config.Target{
		"test.com:5432": makeTarget("test", "test.com:5432"),
	}
	cases := []struct {
		Targets  map[string]config.Target
		Name     string
		Expected error
	}{
		{
			Targets:  nil,
			Name:     "test",
			Expected: discovery.ErrTargetNotFound,
		},
		{
			Targets:  staticTargets,
			Name:     "missing",
			Expected: discovery.ErrTargetNotFound,
		},
	}
	for idx, test := range cases {
		client := NewStaticDiscoveryClient(test.Targets)
		_, err := client.LookupTargetByName(test.Name)
		if test.Expected != err {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, err)
		}
	}
}

func TestStaticDiscoveryClientGetTargets(t *testing.T) {
	staticTargets := map[string]config.Target{
		"test.com:5432": makeTarget("test", "test.com:5432"),
	}
	client := NewStaticDiscoveryClient(staticTargets)
	targets := client.GetTargets()
	if len(targets) != len(staticTargets) {
		t.Fatalf("missing targets")
	}

	for _, target := range targets {
		found, ok := staticTargets[target.Host]
		if !ok {
			t.Fatalf("failed to find target %q", target.Host)
		}
		if found.Host != target.Host || target.Name != found.Name {
			t.Fatalf("found wrong target %q, expected %q", found.Host, target.Host)
		}
	}

}

func makeTarget(name string, host string) config.Target {
	return config.Target{
		Host: host,
		Name: name,
	}
}
