package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

const (
	filterEngine         = "engine"
	enginePostgres       = "postgres"
	engineAuroraPostgres = "aurora-postgresql"
)

// DBInstanceResult is wrapper around a DBInstance or error
// as a result of listing RDS Instances
type DBInstanceResult struct {
	Instance types.DBInstance
	Error    error
}

// RDSClient is our wrapper around the RDS library, allows us to
// mock this for testing
type RDSClient interface {
	GetPostgresInstances(ctx context.Context) <-chan DBInstanceResult
	NewAuthToken(ctx context.Context, host, region, user string) (string, error)
	RegionForInstance(inst types.DBInstance) (string, error)
}

type rdsClient struct {
	cfg aws.Config
	svc *rds.Client
}

// NewRDSClient loads AWS Config and creds, and returns an RDS client
func NewRDSClient(ctx context.Context) (RDSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &rdsClient{cfg: cfg, svc: rds.NewFromConfig(cfg)}, nil
}

// GetPostgresInstances grabs all db instances filtered by engine "postgres" and publishes
// them to the result channel
func (r *rdsClient) GetPostgresInstances(ctx context.Context) <-chan DBInstanceResult {
	resChan := make(chan DBInstanceResult, 1)
	go func() {
		defer close(resChan)
		paginator := r.rdsPaginator([]types.Filter{
			{
				Name:   strPtr(filterEngine),
				Values: []string{enginePostgres, engineAuroraPostgres},
			},
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				resChan <- DBInstanceResult{Error: err}
				return
			}
			for _, d := range page.DBInstances {
				resChan <- DBInstanceResult{Instance: d}
			}
		}
	}()
	return resChan
}

func (r *rdsClient) rdsPaginator(filters []types.Filter) (paginator *rds.DescribeDBInstancesPaginator) {
	paginator = rds.NewDescribeDBInstancesPaginator(r.svc, &rds.DescribeDBInstancesInput{
		Filters: filters,
	}, func(o *rds.DescribeDBInstancesPaginatorOptions) {
		o.Limit = 100
	})
	return
}

func (r *rdsClient) NewAuthToken(ctx context.Context, host, region, user string) (string, error) {
	return auth.BuildAuthToken(ctx, host, region, user, r.cfg.Credentials)
}

func (r *rdsClient) RegionForInstance(inst types.DBInstance) (string, error) {
	arn, err := arn.Parse(*inst.DBInstanceArn)
	if err != nil {
		return "", err
	}
	return arn.Region, nil
}

func strPtr(val string) *string {
	return &val
}
