---
sidebar_position: 4
---

# Configuration YAML Structure

Every configuration needs a YAML containing required and optional content.

A minimal viable config needs to look like this:

```yaml
config:
    - {config name} : "{path of config json template}"

{config name}:
    - name: "{a unique name}"
```

e.g. in `projects/infrastructure/alerting-profile/profiles.yaml`
```yaml
config:
  - profile: "projects/infrastructure/alerting-profile/profile.json"

profile:
  - name: "profile-name"
[...]
```

Every config needs to provide a name for unique identification, omitting the name variable or using a duplicate name will result in a validation / deployment error.

Any defined `{config name}` represents a variable that can then be used in a [JSON template](#config-json-templates), and will be resolved and inserted into the config before deployment to Dynatrace.

e.g. `projects/infrastructure/alerting-profile/profiles.yaml` defines a `name`:
```yaml
[...]
profile:
  - name: "EXAMPLE Infrastructure"
[...]
```

Which is then used in `projects/infrastructure/alerting-profile/profile.json` as `{{.name}}`.

### Skip configuration deployment

To skip configuration from deploying you can use predefined `skipDeployment` parameter. You can skip deployment of the whole configuration:

```yaml
my-config:
  - name: "My config"
  - skipDeployment: "true"
```
enable it by default, but skip for environment or group:
```yaml
my-config:
  - name: "My config"
  - skipDeployment: "true"

my-config.development:
  - skipDeployment: "false"
```
or disable it by default and enable only for environment or group:
```yaml
my-config:
  - name: "My config"
  - skipDeployment: "false"

my-config.environment:
  - skipDeployment: "true"
```

### Specific Configuration per Environment or group

Configuration can be overwritten or extended:
* per environment by adding `.{Environment}` configurations
* per group by adding `.{GROUP}` configurations

e.g. `projects/infrastructure/notification/notifications.yaml` defines different recipients for email notifications for each environment via

```yaml
email:
    [...]

email.group:
    [...]

email.environment1:
    [...]

email.environment2:
    [...]

email.environment3:
    [...]
```

Anything in the base `email` configuration is still applied, unless it's re-defined in the `.{GROUP}` or `.{Environment}` config.

**If both environment and group configurations are defined, then environment
is preferred over the group configuration.**

### Referencing other Configurations

In many cases one auto-deployed Dynatrace configuration will depend on another one.

E.g. Where most configurations depend on the management-zone defined in `projects/infrastructure/management-zone`

The tool allows your configuration to reference either the `name` or `id` of the Dynatrace object of another configuration created on the cluster.

To reference these, the dependent `config yaml` can configure a variable of the format

```
{var} : "{name of the referenced configuration}.[id|name]"
```

e.g. `projects/project-name/dashboard/dashboard.yaml` references the management-zone defined by `/projects/infrastructure/management-zone/zone.json` via
```yaml
  - managementZoneId: "projects/infrastructure/management-zone/zone.id"
```

### Referencing other json templates
Json templates are usually defined inside of project configuration and then references in same project:

**testproject/auto-tag/auto-tag.yaml:**
```yaml
config:
  - application-tagging-multiproject: "application-tagging.json"

application-tagging-multiproject:
  - name: "Test Application Multiproject"
```

In this example, `application-tagging.json` is located in `auto-tag` folder of same project and the path to it
can be defined relative to `auto-tag.yaml` file. But, what if you would like to reuse one template defined outside of this project?
 can be defined relative to `auto-tag.yaml` file. But, what if you would like to reuse one template defined outside of this project?
can be defined relative to `auto-tag.yaml` file. But, what if you would like to reuse one template defined outside of this project?
In this case, you need to define a full path of json template:

**testproject/auto-tag/auto-tag.yaml:**
```yaml
config:
  - application-tagging-multiproject: "/path/to/project/auto-tag/application-tagging.json"

application-tagging-multiproject:
  - name: "Test Application Multiproject"
```
This would save us of content duplication and redefining same templates over and over again.

Of course, it is also possible to reuse one template multiple times within one or different yaml file(s):
**testproject/auto-tag/auto-tag.yaml:**
```yaml
config:
  - application-tagging-multiproject: "/path/to/project/auto-tag/application-tagging.json"
  - application-tagging-tesproject: "/path/to/project/auto-tag/application-tagging.json"
  - application-tagging-otherproject: "/path/to/project/auto-tag/application-tagging.json"

application-tagging-multiproject:
  - name: "Test Application Multiproject"
  - param: "Multiproject parameter"

application-tagging-tesproject:
  - name: "Test Application Tesproject"
  - param: "Tesproject parameter"

application-tagging-otherproject:
  - name: "Test Application Otherproject"
  - param: "Otherproject parameter"
```

### Templating of Environment Variables

In addition to the templating of `json` files, where you need to specify the values in the corresponding `yaml` files, its also possible to resolve
environment variables. This can be done in any `json` or `yaml` file using this syntax: `{{.Env.ENV_VAR}}`.

E.g. to resolve the URL of an environment, use the following snippet:

```yaml
development:
    - name: "Dev"
    - env-url: "{{ .Env.DEV_URL }}"
    - env-token-name: "DEV_TOKEN_ENV_VAR"
```

To resolve an environment variable directly in the `json` is also possible. See the following example which sets the value
of an alerting profile from the env var `ALERTING_PROFILE_VALUE`.

```json
{
  "name": "{{ .name }}",
  "rules": [
    {
      "type": "APPLICATION",
      "enabled": true,
      "valueFormat": null,
      "propagationTypes": [],
      "conditions": [
        {
          "key": {
            "attribute": "WEB_APPLICATION_NAME"
          },
          "comparisonInfo": {
            "type": "STRING",
            "operator": "CONTAINS",
            "value": "{{ .Env.ALERTING_PROFILE_VALUE }}",
            "negate": false,
            "caseSensitive": true
          }
        }
      ]
    }
  ]
}
```

**Attention**: Values you pass into configuration via environment variables must not contain `=`.