# RDS Proxy

A two-layer proxy for connecting into RDS postgres databases 
based on IAM authentication. 

This tool allows you to keep your databases firewalled off, 
manage database access through IAM policies, and no developer 
will ever have to share or type a password.

As a side note, this pairs extremely well with a tool like [saml2aws](https://github.com/Versent/saml2aws)
to ensure AWS/database access uses temporary credentials.

## Contributing 

For contributing, see [project page](https://github.com/mothership/rds-auth-proxy).

## Security

The security of this setup depends on the following assumptions:

* Users do not share laptops / leave them unlocked.
* No untrusted process on the client machine, or server can read the 
  memory of the proxy process.
* You have a secure tunnel, or other means of encrypting the connection to 
  to the server-side proxy (VPN, SSH tunnel, k8s port-forward, etc.).
* You have adequate IAM policies, restricting which the 
  roles/databases a developer may use.
