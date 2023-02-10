---
sidebar_position: 2
---


# Environments file

The environments file is a YAML file used to define to which environment(s) to deploy configurations.  

In the file, you declare the environment URL and the name of the environment variable to use for the API token.

Deployment can be done on a single environment or on several environments.

Here is an example of the structure of an environments file: 

```yaml title="environments.yaml"
foo:
    - name: "foo"
    - env-url: "https://foo.example.com"
    - env-token-name: "FOO_TOKEN_ENV_VAR"

bar:
    - name: "bar"
    - env-url: "https://bar.dynatrace-managed.com/e/environmentid"
    - env-token-name: "BAR_TOKEN_ENV_VAR"
```

## Envrionment API Tokens
The `env-token-name` specifies the name of an environment variable from where the access token for your Dynatrace environment will be loaded.

Please follow the instructions of your Operating System or CI/CD tool on how to make the token value available as an environment variable.

> For example on Linux: `export BAR_TOKEN_ENV_VAR=XXXXXXXXXXX`

For details on API Token permissions, see the column for each [type of configuration](configTypes_tokenPermissions)

## Environment Grouping

Environments can also be grouped, but only one group is allowed per environment. Assign environments to groups with `group.environment`:

```yaml title="environments.yaml"
production.foo:
    - name: "foo"
    - env-url: "https://foo.dynatrace.com"
    - env-token-name: "FOO_TOKEN_ENV_VAR"

production.bar:
    - name: "bar"
    - env-url: "https://bar.dynatrace-managed.com/e/id"
    - env-token-name: "BAR_TOKEN_ENV_VAR"
```
