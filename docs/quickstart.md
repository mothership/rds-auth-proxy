# Getting Started 

!> At the moment, `rds-auth-proxy` only support PostgreSQL-flavored RDS instances.

`rds-auth-proxy` is a binary that contains two major components, a server-side proxy, 
and a client-side proxy. This guide takes you through deploying the server-side proxy in 
your cluster, and connecting to it using the client-side proxy.

For more information about the design, see the [architecture](./architecture.md) docs.

## Deploying the Server

In order to deploy the server proxy successfully, you must ensure the following:

1. The proxy will be deployed into a subnet with security group rules that allow access to the RDS instances
2. The proxy has AWS credentials for database discovery

The recommended way to install the proxy is to use our [helm chart](https://github.com/mothership/helm-charts/tree/master/charts/rds-auth-proxy).

### Setting up AWS Permissions 

The server-side proxy needs to be able to look up database instances to validate that it's allowed 
to complete the connection.

In order to do this, it must be able to list RDS instances. An example IAM policy may look like this:

```json
{
   "Version":"2012-10-17",
   "Statement":[
      {
         "Sid":"AllowRDSDescribe",
         "Effect":"Allow",
         "Action": [
            "rds:DescribeDBInstances",
            "rds:ListTagsForResource"
         ],
         "Resource": [
            "arn:aws:rds:*:*:db:*",
            "arn:aws:rds:*:*:pg:*"
        ]
      }
   ]
}
```

You can get more granular with this policy by only allowing certain tags, AWS accounts, etc. Attach 
this policy to the user or role that will be used by the server-side proxy.

### Adding our chart repository 

Use Helm 3 to add the mothership repository: 

```bash
helm repo add mothership https://mothership.github.io/helm-charts/ 
helm repo update
```

### Installing

Start by creating a values file for the helm chart. An example file using [IRSA](https://aws.amazon.com/blogs/opensource/introducing-fine-grained-iam-roles-service-accounts/) 
is provided below. 

A similar approach should also work for [kube2iam](https://github.com/jtblin/kube2iam), or [kiam](https://github.com/uswitch/kiam) 
as well, but the annotations would need to be added to the deployment instead of the service account.

```yaml
# values.yaml
fullnameOverride: "rds-auth-proxy"
deployment:
  # deploy at least two pods
  replicaCount: 2 

# create a service account, and pass AWS credentials using IRSA
serviceAccount:
  create: true
  # IRSA controller will pick up this annotation and inject AWS credentials into the pod.
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/rds-auth-proxy 

proxy:
  # Disable SSL/TLS to the proxy itself, we'll tunnel the connection over a port-forward 
  ssl:
    enabled: false
  # Disable cert manager integration, the proxy will generate a self-signed client 
  # certificate and key
  certManager:
    enabled: false 
  allowedRDSTags:
    - name: "rds_proxy_enabled"
      value: "true"
```

Apply the helm chart with this configuration to the cluster:

```bash
kubectl create namespace rds-auth-proxy 
helm install rds-auth-proxy --namespace rds-auth-proxy mothership/rds-auth-proxy -f values.yaml
```

## Preparing your database

Enable RDS IAM authentication for one of your databases that the server proxy can reach.

```bash
aws rds modify-db-instance \
    --db-instance-identifier {my-db-instance} \
    --apply-immediately \
    --enable-iam-database-authentication
```

Add the tag `rds_proxy_enabled:true` to your database instance.

```bash
aws rds add-tags-to-resource \
    --resource-name {my-db-arn} \
    --tags "[{\"Key\": \"rds_proxy_enabled\",\"Value\": \"true\"}]"
```

### Granting the IAM Role 

Log in to your database and grant the `rds_iam` role to a user that you want to be accessible over
the proxy.

```sql
GRANT rds_iam TO postgres; 
```

### Granting IAM Permissions 

Ensure that you have access to AWS credentials with permissions to log in as that user. Your IAM
policy would need to include a clause like this:

```json
{
   "Version": "2012-10-17",
   "Statement": [
      {
         "Effect": "Allow",
         "Action": [
             "rds-db:connect"
         ],
         "Resource": [
             "arn:aws:rds-db:{region}:{account}:dbuser:{db-resource-id}/{db-user}"
         ]
      }
   ]
}
```

## Setting up the client

### Download the client binary

Check the [release page](https://github.com/mothership/rds-auth-proxy/releases) for the latest 
binaries. Download and install the one for your platform and architecture.

### Create your local config file

Now we need to tell our client proxy about the server proxy. Drop this file at 
`~/.config/rds-auth-proxy/config.yaml`.

```yaml
# ~/.config/rds-auth-proxy/config.yaml
proxy:
  # this is the host/port that psql should connect with
  listen_addr: "0.0.0.0:8001"
  # don't use SSL between local proxy and psql
  ssl:
    enabled: false 
  # only look at rds instances the server proxy can connect to  
  target_acl:
    allowed_rds_tags:
      - name: rds_proxy_enabled
        value: "true"
    blocked_rds_tags: []

upstream_proxies:
  default:
    # configure a kubernetes port-forward tunnel to the in-cluster proxy
    port_forward:
      # context: some-other-kube-context
      # kube_config: /path/to/alternate_config_file
      deployment: rds-auth-proxy
      namespace: rds-auth-proxy 
      local_port: "8000"
      remote_port: "8000"
    ssl:
      # since we disabled SSL on the in-cluster proxy, don't try SSL between 
      # the client proxy and server proxy
      mode: "disable"
```

## Testing your installation

Run the following to start the client proxy:

```bash
rds-auth-proxy client --target {dbinstanceidentifier}
```

In another shell, you should be able to connect:

```
psql -h localhost -p 8001 -U {db-user-with-iam-auth}
```
