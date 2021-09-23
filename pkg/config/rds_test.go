package config_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/mothership/rds-auth-proxy/pkg/aws"
	. "github.com/mothership/rds-auth-proxy/pkg/config"
)

func TestFetchRDSTargets(t *testing.T) {
	cases := []struct {
		Instances []aws.DBInstanceResult
		Config    ConfigFile
		HostMap   map[string]Target
	}{
		// Case 0: Test no allow/block list
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "false"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true"),
				}),
			},
			HostMap: map[string]Target{
				"case-1-1:5000": {},
				"case-1-2:5000": {},
			},
			Config: configFromACL(nil, nil),
		},
		// Case 1: Test allow list, no block list
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "false"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true"),
				}),
			},
			HostMap: map[string]Target{
				"case-1-2:5000": {},
			},
			Config: configFromACL(tags("enabled", "true"), nil),
		},
		// Case 2: Test no allow list, block list
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "false"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true"),
				}),
			},
			HostMap: map[string]Target{
				"case-1-1:5000": {},
			},
			Config: configFromACL(nil, tags("enabled", "true")),
		},
		// Case 3: Test blocklist overrides allow list
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "false"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true"),
				}),
			},
			HostMap: map[string]Target{},
			Config:  configFromACL(tags("enabled", "true"), tags("enabled", "true")),
		},
		// Case 4: Test multi-tags in allow list require all to be met
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "true", "region", "east"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true", "region", "west"),
				}),
			},
			HostMap: map[string]Target{
				"case-1-2:5000": {},
			},
			Config: configFromACL(tags("enabled", "true", "region", "west"), nil),
		},
		// Case 5: Test multi-tags in block list require any to be met
		{
			Instances: []aws.DBInstanceResult{
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-1"),
					Endpoint:             endpoint("case-1-1", 5000),
					TagList:              rdsTags("enabled", "true", "region", "east"),
				}),
				instance(types.DBInstance{
					DBInstanceIdentifier: strPtr("case-1-2"),
					Endpoint:             endpoint("case-1-2", 5000),
					TagList:              rdsTags("enabled", "true", "region", "west"),
				}),
			},
			HostMap: map[string]Target{
				"case-1-1:5000": {},
			},
			Config: configFromACL(nil, tags("enabled", "false", "region", "west")),
		},
	}

	for idx, test := range cases {
		err := RefreshRDSTargets(context.Background(), &test.Config, &mockRDSClient{Return: test.Instances})
		if err != nil {
			t.Fatalf("[Case %d] expected no error, got: %+v", idx, err)
		}

		for name := range test.HostMap {
			if _, ok := test.Config.HostMap[name]; !ok {
				t.Errorf("[Case %d] Expected %q in config hostmap, got: %+v", idx, name, test.Config.HostMap)
			}
		}

	}
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
	for i := 0; i < len(pairs)/2; i += 2 {
		tags = append(tags, types.Tag{Key: &pairs[i], Value: &pairs[i+1]})
	}
	return tags
}

func tags(pairs ...string) []*Tag {
	if len(pairs)%2 != 0 {
		panic(fmt.Errorf("must pass key value pairs to tags"))
	}
	tags := make([]*Tag, 0, len(pairs)/2)
	for i := 0; i < len(pairs)/2; i += 2 {
		tags = append(tags, &Tag{Name: pairs[i], Value: pairs[i+1]})
	}
	return tags
}

func endpoint(host string, port int32) *types.Endpoint {
	return &types.Endpoint{Address: &host, Port: port}
}

func configFromACL(allowed []*Tag, blocked []*Tag) ConfigFile {
	return ConfigFile{
		Targets: map[string]*Target{},
		HostMap: map[string]*Target{},
		Proxy: Proxy{
			SSL: ServerSSL{},
			ACL: ACL{
				AllowedRDSTags: TagList(allowed),
				BlockedRDSTags: TagList(blocked),
			},
		},
	}
}

func strPtr(val string) *string {
	return &val
}
