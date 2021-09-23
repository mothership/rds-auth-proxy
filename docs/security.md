# Security 

Found an issue? Send us an email at [security@mothership.com](mailto:security@mothership.com). 

As this software is in early development, there are no bounties, but you'll get credit and 
have our eternal gratitude!

# Security Model 

The security of `rds-auth-proxy` depends on the following assumptions:

* Users do not share laptops / leave them unlocked.
* No untrusted process is running on the client machine.
* No untrusted process can read the memory of the proxy process
* You have a secure connection between the client and server proxies 
* You have IAM policies restricting which roles/databases a developer may use.

### Users do not share laptops / leave them unlocked

`rds-auth-proxy` has no means of securing access to the proxy based on who is
using the laptop.

### No untrusted process is running on the client machine 

Any process running locally, with network access, can connect to the proxy.

### No untrusted process can read the memory of the proxy process 

In client mode, `rds-auth-proxy` generates database passwords for the currently 
logged in AWS user. In server mode, `rds-auth-proxy` has to pass along the 
password to RDS. 

If an untrusted process can read memory of the proxy, it can read the generated
password. Additionally, as Go is a garbage collected language, it is difficult
to ensure all copies of the password is cleared from memory.

### You have a secure connection between the client and server proxies

The postgres protocol transports passwords in plaintext and as an md5 hash. 
Neither format is suitable for transport over the public internet.  We recommend 
tunneling the protocol over a kubernetes port-forward, or setting up SSL on the 
server-side proxy.

### You have IAM policies restricting which roles/databases a developer may use

The server proxy, by default, will allow connections into any RDS postgres 
database with IAM authentication enabled. Developer access to the databases are 
controlled by their ability to generate temporary passwords on the client, which
in turn is controlled by IAM policies and the AWS credentials they have access to.

# Options for TLS

### Protecting the connection between the client and client proxy 

We support TLS between the client and client proxy, however, the client proxy is
designed to be run on the same machine as the client and we don't recommend 
setting up TLS in that scenario for a few reasons:

1. If an attacker can already listen to local sockets, they can 
   are already in a privileged position, TLS will not help.
2. Self-signed certificates are the only way to do this, but they are harder 
   to manage and distribute securely.

### Protecting the connection between the client and server proxies 

The client proxy connection can be tunneled over a Kubernetes port-forward, and/or
protected with TLS via the postgres protocol. 

While the server proxy supports the postgres over TLS, the postgres protocol has 
it's own handshake prior to upgrading to TLS, making it difficult to work with 
reverse proxies, or ingress resources.

By using a port-forward, we can piggyback off of the encrypted tunnel Kubernetes 
provides. You can still enable TLS between the client and server over the 
port-forward if desired.

### Protecting the connection between the server and database

For RDS instances, we require full verification of the RDS certificate. Our docker 
images are built with the RDS root CA certificates pre-installed. You may bring your
own client certificate, but `rds-auth-proxy` will generate a self-signed client 
certificate if one isn't provided.
