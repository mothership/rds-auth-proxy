# Config File Reference

## Terms

**Targets** - Databases

**Proxy** - The host launched by the rds-auth-proxy binary, can refer to either the client or server proxy.

**Upstream Proxy** - The server proxy. 

## Database Tags

There are a few database tags that can change the behavior of 
`rds-auth-proxy` on the client proxy. 

| Tag | Behavior |
| --- | -------- |
| `rds-auth-proxy:db-name` | Provides the end user a hint about the default database name |
| `rds-auth-proxy:local-port` | Sets the local port used by the client proxy for that database. Having a static local port per database allows developers to share connection configurations for various database tools |

## Client Config

A full example of every option available:

```yaml
# ~/.config/rds-auth-proxy/config.yaml

# Options for the local proxy server
proxy:
  # The listen address of this proxy
  listen_addr: 0.0.0.0:8001
  # SSL/TLS config for the proxy itself. 
  ssl:
    # If set to true, without specifying a server 
    # certificate/private key, will generate a self-signed
    # certificate for localhost 
    enabled: false 

    # Path to a pem-encoded certificate for the proxy
    certificate: ~/.config/rds-auth-proxy/server-cert.pem 
    # Path to a pem-encoded private key for the certificate 
    private_key: ~/.config/rds-auth-proxy/server-key.pem

    # Path to a pem-encoded certificate for upstream connections 
    #
    # If not set, the proxy will generate a self-signed cert for
    # targets that require TLS/SSL. This cert can be overridden
    # on a per-host basis in the target config block below. 
    #
    # This can be set regardless of whether or not you enable 
    # TLS/SSL.
    client_certificate: ~/.config/rds-auth-proxy/client-cert.pem
    # Path to a pem-encoded key for the client certificate
    #
    # Leave unset, unless you're also providing the 
    # client_certificate.
    client_private_key: ~/.config/rds-auth-proxy/client-key.pem

  # Effectively service-discovery for the proxy. These should
  # match the configuration for the upstream proxy. 
  #
  # In the case that you have multiple upstream proxies, this can 
  # be more permissive, but this has the current limitation that 
  # your user must know which upstream handles which database 
  # instances.
  target_acl:
    # RDS instances must have ALL of these tags to be connectable
    # An empty list means ALL instances that the proxy can see are 
    # connectable
    allowed_rds_tags: 
      - name: "rds_proxy_enabled"
        value: "true"  # currently, must be an exact match
    # RDS instances must not have ANY of these tags to be connectable
    # An empty list means ALL instances that the proxy can see are 
    # connectable 
    blocked_rds_tags: 
      - name: "rds_proxy_disabled"
        value: "true"  # currently, must be an exact match

# This is where you can specify upstream proxy settings
upstream_proxies:
  # The 'default' proxy is used when no --proxy-target flag is 
  # passed to the CLI
  default: 
    # You can set up a kubernetes port-forward to the deployment
    # using a port-forward config block.
    #
    # In this case, the host will be set to 0.0.0.0
    port_forward:
      # The kubernetes config file to use when establishing the port-forward
      # If unset, uses ~/.kube/config
      kube_config: ~/.config/kube/kube_config
      # The context to use within the kube config file
      context: development
      # The name of your server proxy deployment
      deployment: rds-auth-proxy
      # The namespace of your server proxy
      namespace: rds-auth-proxy 
      # Optional, the local port for the port-forward tunnel
      # if not specified, a random unused port will be used. If you have
      # multiple upstream proxies, leave this unset!
      local_port: 8000
      # The remote port of the proxy 
      remote_port: 8000
    ssl:
      # You can enable SSL over a port-forward, but it's not required
      # as the port-forward is over a TLS connection. 
      mode: "disable" # options are "disable", "verify-full", "verify-ca", or "require"
  # Additional upstream proxies can be specified as arbitrary keys in
  # the block 
  with_ssl:
    host: example.com:8000
    ssl:
      # Expects the server certificate to be signed by a CA in the system trust store
      # and that the common name of the certificate matches the hostname
      mode: "verify-full"
      # Optionally provide a root CA that the certifiate chain must validate up to, 
      # rather than the system trust store.
      root_certificate: ~/.config/rds-auth-proxy/root-ca.pem
      # Path to a pem encoded client certificate that should be used instead of the 
      # proxies default client certificate for this host
      client_cert: ~/.config/rds-auth-proxy/my-client-cert.pem 
      # Path to the pem encoded private key for the certificate 
      client_private_key: ~/.config/rds-auth-proxy/my-client-key.pem 

# This is where you can specify SSL settings for the upstream
# (non-RDS) databases 
#
# RDS databases are discovered/added automatically at runtime.
targets:
  in-cluster-postgres:
    # This should be the in-cluster hostname / port that the server-proxy 
    # will use.
    host: postgres:5432
```

## Server Config

A full example of every option available:

```yaml
# /etc/rds-auth-proxy/config.yaml 

# Options for the local proxy server
proxy:
  # The listen address of this proxy
  listen_addr: 0.0.0.0:8000
  # SSL/TLS config for the proxy itself. 
  ssl:
    # If set to true, without specifying a server 
    # certificate/private key, will generate a self-signed
    # certificate for localhost 
    enabled: false 

    # Path to a pem-encoded certificate for the proxy
    certificate: /etc/rds-auth-proxy/server-cert.pem 
    # Path to a pem-encoded private key for the certificate 
    private_key: /etc/rds-auth-proxy/server-key.pem

    # Path to a pem-encoded certificate for upstream connections 
    #
    # If not set, the proxy will generate a self-signed cert for
    # targets that require TLS/SSL. This cert can be overridden
    # on a per-host basis in the target config block below. 
    #
    # This can be set regardless of whether or not you enable 
    # TLS/SSL.
    client_certificate: /etc/rds-auth-proxy/client-cert.pem
    # Path to a pem-encoded key for the client certificate
    #
    # Leave unset, unless you're also providing the 
    # client_certificate.
    client_private_key: /etc/rds-auth-proxy/client-key.pem

  # Effectively service-discovery for the proxy. Before making an
  # outbound connection, the proxy will check and verify that it
  # knows the host has one of these tags, or is specified in the
  # target list below.
  target_acl:
    # RDS instances must have ALL of these tags to be connectable
    # An empty list means ALL instances that the proxy can see are 
    # connectable
    allowed_rds_tags: 
      - name: "rds_proxy_enabled"
        value: "true"  # currently, must be an exact match
    # RDS instances must not have ANY of these tags to be connectable
    # An empty list means ALL instances that the proxy can see are 
    # connectable 
    blocked_rds_tags: 
      - name: "rds_proxy_disabled"
        value: "true"  # currently, must be an exact match

# This is where you can specify SSL settings for the upstream
# databases 
#
# RDS databases are discovered/added automatically at runtime, if you
# add them yourself, you can override SSL settings.
targets:
  in-cluster-postgres:
    # This should be the in-cluster hostname / port that the server-proxy 
    # will use.
    host: postgres:5432
  overriden-rds-ssl:
    host: test-rds.aws.com:5432
    ssl:
      mode: "verify-full"
      # Path to a pem encoded client certificate that should be used instead of the 
      # proxies default client certificate for this host
      client_cert: /etc/rds-auth-proxy/my-client-cert.pem 
      # Path to the pem encoded private key for the certificate 
      client_private_key: /etc/rds-auth-proxy/my-client-key.pem 
```
