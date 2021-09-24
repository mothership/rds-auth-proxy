# RDS Proxy

A two-layer proxy for connecting into RDS postgres databases 
based on IAM authentication. 

This tool allows you to keep your databases firewalled off, 
manage database access through IAM policies, and no developer 
will ever have to share or type a password.

As a side note, this pairs extremely well with a tool like [saml2aws](https://github.com/Versent/saml2aws)
to ensure AWS/database access uses temporary credentials.

## General Usage

General documentation is available on our [project site](https://mothership.github.io/rds-auth-proxy/).

## Design 

One proxy is run in your VPC subnet that can reach your RDS instances,
the other on your client machine (dev laptop, etc.) with access to 
aws credentials.

The client proxy is responsible for picking a host (RDS instance), and 
generating a temporary password based on the local IAM identity. The
client proxy injects the host and password into the postgres startup 
message as additional parameters. 

![Client startup flow](./docs/images/rds-proxy-client-startup-flow.png)

The server proxy accepts a connection from the client proxy, and 
unpacks the host and password parameters. It then opens a connection 
to the RDS database and intercepts the authentication request. It then 
passes along the password it received from the client, and forwards the 
result to the client.

![Auth overview](./docs/images/rds-proxy-auth-flow.png)

## Security

The security of this setup depends on the following assumptions:

* Users do not share laptops / leave them unlocked.
* No untrusted process on the client machine, or server can read the 
  memory of the proxy process.
* You have a secure tunnel, or other means of encrypting the connection to 
  the server-side proxy (VPN, SSH tunnel, k8s port-forward, etc.).
* You have adequate IAM policies, restricting which the 
  roles/databases a developer may use.


## Releasing

CI handles building binaries and images on tag events. 

To create a release, start with a dry-run on the main branch:

```bash
git checkout main
./build/release.sh --dry-run
```

Ensure that the changelog looks as expected, then run it for real:

```bash
./build/release.sh
```

