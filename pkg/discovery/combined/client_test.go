package combined_test

import (
	"testing"

	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	. "github.com/mothership/rds-auth-proxy/pkg/discovery/combined"
	"github.com/mothership/rds-auth-proxy/pkg/discovery/static"
)

func TestCombinedDiscoveryClientHostLookupFailures(t *testing.T) {
	staticOne := makeStatic(makeTarget("db-1", "db-1:5432"))
	staticTwo := makeStatic(makeTarget("db-2", "db-2:5432"), makeTarget("db-3", "db-3:5432"))
	cases := []struct {
		Clients  []discovery.Client
		Host     string
		Expected error
	}{
		{
			Clients:  nil,
			Host:     "db-1:5432",
			Expected: discovery.ErrTargetNotFound,
		},
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Host:     "db-4:5432",
			Expected: discovery.ErrTargetNotFound,
		},
	}
	for idx, test := range cases {
		client := NewCombinedDiscoveryClient(test.Clients)
		_, err := client.LookupTargetByHost(test.Host)
		if test.Expected != err {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, err)
		}
	}
}

func TestCombinedDiscoveryClientHostLookupSuccess(t *testing.T) {
	staticOne := makeStatic(makeTarget("db-1", "db-1:5432"))
	staticTwo := makeStatic(makeTarget("db-2", "db-2:5432"), makeTarget("db-3", "db-3:5432"))
	cases := []struct {
		Clients  []discovery.Client
		Host     string
		Expected config.Target
	}{
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Host:     "db-1:5432",
			Expected: ensureTarget(staticOne.LookupTargetByHost("db-1:5432")),
		},
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Host:     "db-2:5432",
			Expected: ensureTarget(staticTwo.LookupTargetByHost("db-2:5432")),
		},
	}
	for idx, test := range cases {
		client := NewCombinedDiscoveryClient(test.Clients)
		target, err := client.LookupTargetByHost(test.Host)
		if err != nil {
			t.Fatalf("[Case %d] unexpected error: %s", idx, err)
		}
		if test.Expected != target {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestCombinedDiscoveryClientNameLookupSuccess(t *testing.T) {
	staticOne := makeStatic(makeTarget("db-1", "db-1:5432"))
	staticTwo := makeStatic(makeTarget("db-2", "db-2:5432"), makeTarget("db-3", "db-3:5432"))
	cases := []struct {
		Clients  []discovery.Client
		Name     string
		Expected config.Target
	}{
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Name:     "db-1",
			Expected: ensureTarget(staticOne.LookupTargetByHost("db-1:5432")),
		},
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Name:     "db-3",
			Expected: ensureTarget(staticTwo.LookupTargetByHost("db-3:5432")),
		},
	}
	for idx, test := range cases {
		client := NewCombinedDiscoveryClient(test.Clients)
		target, err := client.LookupTargetByName(test.Name)
		if err != nil {
			t.Fatalf("[Case %d] unexpected error: %s", idx, err)
		}
		if test.Expected != target {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestCombinedDiscoveryClientNameLookupFailures(t *testing.T) {
	staticOne := makeStatic(makeTarget("db-1", "db-1:5432"))
	staticTwo := makeStatic(makeTarget("db-2", "db-2:5432"), makeTarget("db-3", "db-3:5432"))
	cases := []struct {
		Clients  []discovery.Client
		Name     string
		Expected error
	}{
		{
			Clients:  nil,
			Name:     "db-1",
			Expected: discovery.ErrTargetNotFound,
		},
		{
			Clients:  []discovery.Client{staticOne, staticTwo},
			Name:     "db-4",
			Expected: discovery.ErrTargetNotFound,
		},
	}
	for idx, test := range cases {
		client := NewCombinedDiscoveryClient(test.Clients)
		_, err := client.LookupTargetByName(test.Name)
		if test.Expected != err {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, err)
		}
	}
}

func TestCombinedDiscoveryClientGetTargets(t *testing.T) {
	staticOne := makeStatic(makeTarget("db-1", "db-1:5432"))
	staticTwo := makeStatic(makeTarget("db-2", "db-2:5432"), makeTarget("db-3", "db-3:5432"))
	client := NewCombinedDiscoveryClient([]discovery.Client{staticOne, staticTwo})
	targets := client.GetTargets()
	if len(targets) != len(staticOne.GetTargets())+len(staticTwo.GetTargets()) {
		t.Fatalf("missing targets")
	}

	for _, target := range targets {
		found, err := client.LookupTargetByHost(target.Host)
		if err != nil {
			t.Fatalf("unexpected error %+v", err)
		}
		if found != target {
			t.Fatalf("found wrong target: %+v, expected %+v", found, target)
		}
	}
}

func makeStatic(targets ...config.Target) discovery.Client {
	hostMap := map[string]config.Target{}
	for _, target := range targets {
		hostMap[target.Host] = target
	}
	return static.NewStaticDiscoveryClient(hostMap)
}

func makeTarget(name string, host string) config.Target {
	return config.Target{
		Host: host,
		Name: name,
	}
}

func ensureTarget(target config.Target, err error) config.Target {
	if err != nil {
		panic(err)
	}
	return target
}
