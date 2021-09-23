# Unique Local Ports Per Database

In some cases, you may want a unique local port per database instead of 
the default listening port for the proxy. 

Maybe you want your staging environment use the local port range 
`54000-54999`, and your production environment to use local ports 
`55000-55999`. Maybe you want to save and share connection details across
various database GUIs or other tooling.

Whatever the case, we can do that with the tag `rds-auth-proxy:local-port`.

## Adding the Tag

```bash
aws rds add-tags-to-resource \
    --resource-name {your-db-arn} \
    --tags "[{\"Key\": \"rds-auth-proxy:local-port\",\"Value\": \"54000\"}]"
```

## Try it out

Now, when any of your developers run the client proxy, they should see the local proxy
boot on port `54000`.

```bash
rds-auth-proxy client --target {my-db-identifier}
```
