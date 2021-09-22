# Architecture 

`rds-auth-proxy` is a binary containing two different proxies.
One proxy is run in a VPC subnet that can reach your RDS instances,
the other on your client machine (dev laptop, etc.) with access to 
aws credentials.

The client proxy is responsible for picking a host (RDS instance), and 
generating a temporary password using the local IAM identity. The
client proxy injects the desired host and password into the postgres 
startup message as additional parameters. 

![Client startup flow](./images/rds-proxy-client-startup-flow.png)

The server proxy accepts a connection from the client proxy, and 
unpacks the host and password parameters. The server proxy checks 
that it's allowed to connect to the postgres database, based on 
the set of allowed/blocked tags specified in the config file.

The server proxy then opens a connection to the RDS database and intercepts 
the authentication request. It passes along the password it received from 
the client, and forwards the result to the client. 

![Auth overview](./images/rds-proxy-auth-flow.png)

After successful auth, all messages are proxied transparently between the 
client and database.

## Protecting the connection between the client and client proxy 

We support TLS between the client and client proxy, however, the client proxy is
designed to be run on the same machine as the client and we don't recommend 
setting up TLS in that scenario for a few reasons:

1. If an attacker can already listen to local sockets, they can 
   are already in a privileged position, TLS will not help.
2. Self-signed certificates are the only way to go, but harder to manage 
   and distribute securely.


## Protecting the connection between the client and server proxies 

The client proxy connection can be tunneled over a Kubernetes port-forward, and/or
protected with TLS via the postgres protocol. 

While the server proxy supports the postgres over TLS, the postgres protocol has 
it's own handshake prior to upgrading to TLS, making it difficult to work with 
reverse proxies, or ingress resources.

By using a port-forward, we can piggyback off of the encrypted tunnel Kubernetes 
is providing. You can still enable TLS between the client and server over the 
port-forward if desired.


## Protecting the connection between the server and database

For RDS instances, we require full verification of the RDS certificate. Our docker images
are built with the RDS root CA certificates pre-installed.
