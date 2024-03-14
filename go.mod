module github.com/mothership/rds-auth-proxy

go 1.14

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/aws/aws-sdk-go-v2 v1.9.1
	github.com/aws/aws-sdk-go-v2/config v1.8.2
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.1.7
	github.com/aws/aws-sdk-go-v2/service/rds v1.9.0
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.8.1
	go.uber.org/zap v1.19.1
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
)
