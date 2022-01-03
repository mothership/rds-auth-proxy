package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
)

// RedshiftClusterResult is wrapper around a DBInstance or error
// as a result of listing Redshift cluster
type RedshiftClusterResult struct {
	Instance types.Cluster
	Error    error
}

// RedshiftClient is our wrapper around the Redshift library, allows us to
// mock this for testing
type RedshiftClient interface {
	GetRedshiftInstances(ctx context.Context) <-chan RedshiftClusterResult
	NewAuthToken(ctx context.Context, clusterId, region, user string) (string, error)
	RegionForInstance(inst types.Cluster) (string, error)
}

type redshiftClient struct {
	cfg aws.Config
	svc *redshift.Client
}

// NewRedshiftClient loads AWS Config and creds, and returns an Redshift client
func NewRedshiftClient(ctx context.Context) (*redshiftClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &redshiftClient{cfg: cfg, svc: redshift.NewFromConfig(cfg)}, nil
}

// GetPostgresInstances grabs all db instances filtered by engine "postgres" and publishes
// them to the result channel
func (r *redshiftClient) GetRedshiftInstances(ctx context.Context) <-chan RedshiftClusterResult {
	resChan := make(chan RedshiftClusterResult, 1)
	go func() {
		defer close(resChan)
		paginator := r.redshiftPaginator()
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				resChan <- RedshiftClusterResult{Error: err}
				return
			}
			for _, d := range page.Clusters {
				resChan <- RedshiftClusterResult{Instance: d}
			}
		}
	}()
	return resChan
}

func (r *redshiftClient) redshiftPaginator() (paginator *redshift.DescribeClustersPaginator) {
	paginator = redshift.NewDescribeClustersPaginator(r.svc, &redshift.DescribeClustersInput{}, func(o *redshift.DescribeClustersPaginatorOptions) {
		o.Limit = 100
	})
	return
}

func (r *redshiftClient) NewAuthToken(ctx context.Context, clusterId, region, user string) (string, error) {
	credentials, err := r.svc.GetClusterCredentials(ctx, &redshift.GetClusterCredentialsInput{
		AutoCreate: aws.Bool(false),
		ClusterIdentifier: strPtr(clusterId),
		DbUser: strPtr(user),
	})

	return *credentials.DbPassword, err
}

func (r *redshiftClient) RegionForInstance(inst types.Cluster) (string, error) {
	arn, err := arn.Parse(*inst.ClusterNamespaceArn)
	if err != nil {
		return "", err
	}
	return arn.Region, nil
}
