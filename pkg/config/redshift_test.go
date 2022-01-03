package config_test

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/mothership/rds-auth-proxy/pkg/aws"
)

type mockRedshiftClient struct {
	Return []aws.RedshiftClusterResult
}

var _ aws.RedshiftClient = (*mockRedshiftClient)(nil)

func (m *mockRedshiftClient) GetRedshiftInstances(ctx context.Context) <-chan aws.RedshiftClusterResult {
	retChan := make(chan aws.RedshiftClusterResult, 1)
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

func (m *mockRedshiftClient) NewAuthToken(ctx context.Context, clusterId, region, user string) (string, error) {
	return "", nil
}

func (m *mockRedshiftClient) RegionForInstance(d types.Cluster) (string, error) {
	return "us-west-2", nil
}