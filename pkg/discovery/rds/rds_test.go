package rds_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/mothership/rds-auth-proxy/pkg/aws"
	"github.com/mothership/rds-auth-proxy/pkg/config"
	"github.com/mothership/rds-auth-proxy/pkg/discovery"
	. "github.com/mothership/rds-auth-proxy/pkg/discovery/rds"
)

func TestRefreshBehaviorWithACLFailures(t *testing.T) {
	instances := []aws.DBInstanceResult{
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-1"),
			Endpoint:             endpoint("db-1", 5000),
			TagList:              rdsTags("enabled", "false"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-2"),
			Endpoint:             endpoint("db-2", 5000),
			TagList:              rdsTags("enabled", "true"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-3"),
			Endpoint:             endpoint("db-3", 5000),
			TagList:              rdsTags("region", "east", "enabled", "true"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-4"),
			Endpoint:             endpoint("db-4", 5000),
			TagList:              rdsTags("region", "west", "enabled", "true"),
		}),
	}

	cases := []struct {
		Config   config.ConfigFile
		Host     string
		Expected error
	}{
		// Case 0: Test no allow/block list
		{
			Config:   configFromACL(nil, nil),
			Host:     "db-2:5000",
			Expected: nil,
		},
		// Case 1: Test allow list, no block list
		{
			Config:   configFromACL(tags("enabled", "true"), nil),
			Host:     "db-1:5000",
			Expected: discovery.ErrTargetNotFound,
		},
		// Case 2: Test no allow list, block list
		{
			Config:   configFromACL(nil, tags("enabled", "true")),
			Host:     "db-1:5000",
			Expected: nil,
		},
		// Case 3: Test blocklist overrides allow list
		{
			Config:   configFromACL(tags("enabled", "true"), tags("enabled", "true")),
			Host:     "db-1:5000",
			Expected: discovery.ErrTargetNotFound,
		},
		// Case 4: Test multi-tags in allow list require all to be met
		{
			Config:   configFromACL(tags("enabled", "true", "region", "west"), nil),
			Host:     "db-3:5000",
			Expected: discovery.ErrTargetNotFound,
		},
		// Case 5: Test multi-tags in block list require any to be met
		{
			Config:   configFromACL(nil, tags("enabled", "false", "region", "west")),
			Host:     "db-4:5000",
			Expected: discovery.ErrTargetNotFound,
		},
	}

	for idx, test := range cases {
		client := NewRdsDiscoveryClient(&mockRDSClient{Return: instances}, &test.Config)
		if err := client.Refresh(context.Background()); err != nil {
			t.Fatalf("[Case %d] expected no error, got: %+v", idx, err)
		}
		_, err := client.LookupTargetByHost(test.Host)
		if err != test.Expected {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, err)
		}
	}
}

func TestRefreshBehaviorWithACLSuccesses(t *testing.T) {
	instances := []aws.DBInstanceResult{
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-1"),
			Endpoint:             endpoint("db-1", 5000),
			TagList:              rdsTags("enabled", "false"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-2"),
			Endpoint:             endpoint("db-2", 5000),
			TagList:              rdsTags("enabled", "true"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-3"),
			Endpoint:             endpoint("db-3", 5000),
			TagList:              rdsTags("region", "east", "enabled", "true"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-4"),
			Endpoint:             endpoint("db-4", 5000),
			TagList:              rdsTags("region", "west", "enabled", "true"),
		}),
	}
	cases := []struct {
		Config   config.ConfigFile
		Host     string
		Expected config.Target
	}{
		// Case 0: Test no allow/block list
		{
			Config:   configFromACL(nil, nil),
			Host:     "db-2:5000",
			Expected: config.Target{Name: "db-2"},
		},
		// Case 1: Test block list still allows non-blocked dbs
		{
			Config:   configFromACL(nil, tags("enabled", "true")),
			Host:     "db-1:5000",
			Expected: config.Target{Name: "db-1"},
		},
		// Case 2: Test multi-tags in block list doesn't affect other dbs
		{
			Config:   configFromACL(nil, tags("enabled", "false", "region", "west")),
			Host:     "db-3:5000",
			Expected: config.Target{Name: "db-3"},
		},
	}
	for idx, test := range cases {
		client := NewRdsDiscoveryClient(&mockRDSClient{Return: instances}, &test.Config)
		if err := client.Refresh(context.Background()); err != nil {
			t.Fatalf("[Case %d] expected no error, got: %+v", idx, err)
		}
		target, err := client.LookupTargetByHost(test.Host)
		if err != nil {
			t.Fatalf("[Case %d] got unexpected error: %s", idx, err)
		}
		if target.Name != test.Expected.Name {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestGetTargetByNameSuccesses(t *testing.T) {
	instances := []aws.DBInstanceResult{
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-1"),
			Endpoint:             endpoint("db-1", 5000),
			TagList:              rdsTags("enabled", "false"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-2"),
			Endpoint:             endpoint("db-2", 5000),
			TagList:              rdsTags("enabled", "true"),
		}),
	}
	cases := []struct {
		Name     string
		Expected config.Target
	}{
		{
			Name:     "db-2",
			Expected: config.Target{Name: "db-2"},
		},
		{
			Name:     "db-1",
			Expected: config.Target{Name: "db-1"},
		},
	}
	for idx, test := range cases {
		config := configFromACL(nil, nil)
		client := NewRdsDiscoveryClient(&mockRDSClient{Return: instances}, &config)
		if err := client.Refresh(context.Background()); err != nil {
			t.Fatalf("[Case %d] expected no error, got: %+v", idx, err)
		}
		target, err := client.LookupTargetByName(test.Name)
		if err != nil {
			t.Fatalf("[Case %d] got unexpected error: %s", idx, err)
		}
		if target.Name != test.Expected.Name {
			t.Errorf("[Case %d] expected %+v. Got %+v.", idx, test.Expected, target)
		}
	}
}

func TestGetTargetByNameFailures(t *testing.T) {
	instances := []aws.DBInstanceResult{
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-1"),
			Endpoint:             endpoint("db-1", 5000),
			TagList:              rdsTags("enabled", "false"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-2"),
			Endpoint:             endpoint("db-2", 5000),
			TagList:              rdsTags("enabled", "true"),
		}),
	}
	cases := []struct {
		Name     string
		Expected error
	}{
		{
			Name:     "db-3",
			Expected: discovery.ErrTargetNotFound,
		},
	}
	for idx, test := range cases {
		config := configFromACL(nil, nil)
		client := NewRdsDiscoveryClient(&mockRDSClient{Return: instances}, &config)
		if err := client.Refresh(context.Background()); err != nil {
			t.Fatalf("[Case %d] expected no error, got: %+v", idx, err)
		}
		_, err := client.LookupTargetByName(test.Name)
		if err != test.Expected {
			t.Fatalf("[Case %d] got %+v, expected error: %s", idx, err, test.Expected)
		}
	}
}

func TestRdsDiscoveryClientGetTargets(t *testing.T) {
	instances := []aws.DBInstanceResult{
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-1"),
			Endpoint:             endpoint("db-1", 5000),
			TagList:              rdsTags("enabled", "false"),
		}),
		instance(types.DBInstance{
			DBInstanceIdentifier: strPtr("db-2"),
			Endpoint:             endpoint("db-2", 5000),
			TagList:              rdsTags("enabled", "true"),
		}),
	}
	config := configFromACL(nil, nil)
	client := NewRdsDiscoveryClient(&mockRDSClient{Return: instances}, &config)
	if err := client.Refresh(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %+v", err)
	}
	targets := client.GetTargets()
	if len(targets) != len(instances) {
		t.Fatalf("missing targets")
	}

	// TODO: sort and test that each instance was present, currently a low quality test
}

type mockRDSClient struct {
	Return []aws.DBInstanceResult
}

var _ aws.RDSClient = (*mockRDSClient)(nil)

func (m *mockRDSClient) GetPostgresInstances(ctx context.Context) <-chan aws.DBInstanceResult {
	retChan := make(chan aws.DBInstanceResult, 1)
	go func() {
		defer close(retChan)
		for _, r := range m.Return {
			retChan <- r
			if r.Error != nil {
				return
			}
		}
	}()
	return retChan
}

func (m *mockRDSClient) NewAuthToken(ctx context.Context, host, region, user string) (string, error) {
	return "", nil
}

func (m *mockRDSClient) RegionForInstance(d types.DBInstance) (string, error) {
	return "us-west-2", nil
}

func instance(inst types.DBInstance) aws.DBInstanceResult {
	inst.IAMDatabaseAuthenticationEnabled = true
	return aws.DBInstanceResult{Instance: inst}
}

func rdsTags(pairs ...string) []types.Tag {
	if len(pairs)%2 != 0 {
		panic(fmt.Errorf("must pass key value pairs to rdsTags"))
	}
	tags := make([]types.Tag, 0, len(pairs)/2)
	for i := 0; i < len(pairs)/2+1; i += 2 {
		tags = append(tags, types.Tag{Key: &pairs[i], Value: &pairs[i+1]})
	}
	return tags
}

func tags(pairs ...string) []*config.Tag {
	if len(pairs)%2 != 0 {
		panic(fmt.Errorf("must pass key value pairs to tags"))
	}
	tags := make([]*config.Tag, 0, len(pairs)/2)
	for i := 0; i < len(pairs)/2+1; i += 2 {
		tags = append(tags, &config.Tag{Name: pairs[i], Value: pairs[i+1]})
	}
	return tags
}

func endpoint(host string, port int32) *types.Endpoint {
	return &types.Endpoint{Address: &host, Port: port}
}

func configFromACL(allowed []*config.Tag, blocked []*config.Tag) config.ConfigFile {
	return config.ConfigFile{
		Targets: map[string]*config.Target{},
		Proxy: config.Proxy{
			SSL: config.ServerSSL{},
			ACL: config.ACL{
				AllowedRDSTags: config.TagList(allowed),
				BlockedRDSTags: config.TagList(blocked),
			},
		},
	}
}

func strPtr(val string) *string {
	return &val
}
