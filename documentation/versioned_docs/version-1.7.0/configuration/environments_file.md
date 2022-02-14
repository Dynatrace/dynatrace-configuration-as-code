---
sidebar_position: 2
---


# Environments file

The environments file is a YAML file used to define to which environment(s) to deploy configurations.  

In the file, you declare the environment url and the name of the environment variable to use for the API token.

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