# RDS Proxy

## General Usage

For contributing, see [contribution guide](https://github.com/mothership/rds-auth-proxy).

## Security

The security of this setup depends on the following assumptions:

* Users do not share laptops / leave them unlocked.
* No untrusted process on the client machine, or server can read the 
  memory of the proxy process.
* You have a secure tunnel, or other means of encrypting the connection to 
  to the server-side proxy (VPN, SSH tunnel, k8s port-forward, etc.).
* You have adequate IAM policies, restricting which the 
  roles/databases a developer may use.