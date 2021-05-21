---
sidebar_position: 2
---


# Environments file

Environments are defined in the environments.yaml consisting of the environment url and the name of the environment variable to use for the API token.

Deployment could be done a single environment or several environments defined in the environments.yaml file.

A environment yaml file structure is of the form:

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

Environments can also be grouped. Only one group per environment is allowed. Assign environments to groups with `group.environment`:

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