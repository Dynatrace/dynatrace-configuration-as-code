# Dynatrace Monitoring as Code

This tool automates deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

**For release notes please see [RELEASE_NOTES.md](./RELEASE_NOTES.md)**

**If you wish to contribute please read [CONTRIBUTING.md](./CONTRIBUTING.md)**

**Table of Contents**
> Probably you're most interested in [Using Monitoring as Code (monaco) Tool](#using-monitoring-as-code-tool)
and [Configuration Structure](#configuration-structure)

- [Monitoring as Code Tool](#monitoring-as-code-tool)
  - [Using Monitoring as Code (monaco) Tool](#using-monitoring-as-code-tool)
    - [Commands (CLI)](#commands-cli)
      - [Dry Run (Validating Configuration)](#dry-run-validating-configuration)
    - [Deploying Configuration to Dynatrace](#deploying-configuration-to-dynatrace)
      - [Running The Tool](#running-the-tool)
      - [Environments file](#environments-file)
  - [Configuration Structure](#configuration-structure)
    - [Projects](#projects)
    - [Config JSON Templates](#config-json-templates)
      - [Things you should know](#things-you-should-know)
        - [Dashboard JSON](#dashboard-json)
        - [Calculated log metrics JSON](#calculated-log-metrics-json)
        - [Conditional naming JSON](#conditional-naming-json)
    - [Configuration Types / APIs](#configuration-types--apis)
      - [Supported Configuration Types](#supported-configuration-types)
    - [Configuration YAML Structure](#configuration-yaml-structure)
    - [Skip configuration deployment](#skip-configuration-deployment)
    - [Specific Configuration per Environment or Group](#specific-configuration-per-environment-or-group)
    - [Referencing other JSON templates](#referencing-other-json-templates)
    - [Referencing other json templates](#referencing-other-json-templates)
    - [Plugin Configuration](#plugin-configuration)
    - [Delete Configuration](#delete-configuration)

---

## Using Monitoring as Code Tool

Download the latest [release](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/tag/v1.0.0) of the tool.


### Commands (CLI)

Monitoring as Code is controlled via command-line interface.

The tool is a single command line application that takes required and optional arguments via flags such as `--environments`, `--project` or `--dry-run`.

The tool always depends on a config folder where all configuration projects are stored, possibly with further project subfolders.
If nothing is supplied the current working dir is used.

For deploying a specific project inside a root config folder, the tool could be run as:

`monaco -p="project-folder" -e="environments.yaml" projects-root-folder`

In this case the **project** is within the **projects-root-folder**.

Multiple projects can be specified as well:

`-p="project1,project2,project3"`

The supported flags are described below:

```
$ ./monaco
Please provide environments yaml with -e/--environments!
Usage of arguments:
  -d    Set dry-run flag to just validate configurations instead of deploying. (shorthand)
  -dry-run
        Set dry-run flag to just validate configurations instead of deploying.
  -p string
        Project configuration to deploy. Also deploys any dependent configuration. (shorthand)
  -project string
        Project configuration to deploy. Also deploys any dependent configuration.
   -specific-environment string
        Specifc environment (from list) to deploy to.
  -se string
        Specifc environment (from list) to deploy to. (shorthand)
  -e string
        Mandatory yaml file containing environments to deploy to. (shorthand)
  -environments string
        Mandatory yaml file containing environments to deploy to.
  -v    Set verbose flag to enable debug logging. (shorthand)
  -verbose
        Set verbose flag to enable debug logging.
```

#### Dry Run (Validating Configuration)

The tool allows for basic validation of your config by performing a dry run.

It will check whether your Dynatrace config files are valid JSON, and whether your tool configuration yaml files can be parsed and used.

To validate the configuration execute `monaco -dry-run` on a yaml file as show here:
```
./monaco -dry-run --environments=project/sub-project/my-environments.yaml
2020/06/16 16:22:30 monaco v1.0.0
2020/06/16 16:22:30 Reading projects...
2020/06/16 16:22:30 Sorting projects...
...
2020/06/16 16:22:30 Config validation SUCCESSFUL
```

### Deploying Configuration to Dynatrace

The tool allows for deploying a configuration or a set of configurations in the form of `project(s)`.
A project is a folder containing files that define configurations to be deployed to a environment or a group of environments.
This is done by passing the `--project` flag (or `-p` for short).

#### Running The Tool

Below you find a few samples on how to run the tool to deploy your configurations:

```
monaco -e=environments.yaml (deploy all projects in the current folder to all environments)

monaco -e=environments.yaml -p="project" projects-root-folder (deploy projects-root-folder/project and any projects in projects-root-folder it depends on to all environments)

monaco -e=environments.yaml -p="projectA, projectB" projects-root-folder (deploy projects-root-folder/projectA, projectB and dependencies to all environments)

monaco -e=environments.yaml -st dev (deploy all projects in the current folder to the "dev" environment defined in environments.yaml)
```

If `project` contains additional sub-projects, then all projects are deployed recursively.

If `project` depends on different projects under the same root, those are also deployed.

Multiple projects could be specified by `-p="projectA, projectB, projectC/subproject"`

To deploy configuration the tool will need a valid API Token(s) for the given environments defined as environment variables - you can define the name of that env var in the environments file.

To deploy to 1 specific environment within a `environments.yaml` file, the `-specific-environment` or `-se` flag can be passed:

```
monaco -t=environments.yaml -se=prod-saas -p="prod-saas" cluster
```


#### Environments file
environments are defined in the `environments.yaml` consisting of the environment url and the name of the environment variable to use for the API token.

Deployment could be done a single environment or several environments defined in the `environments.yaml` file.

A environment yaml file structure is of the form:

```
foo:
    - name: "foo"
    - environment-url: "https://foo.example.com"
    - env-token-name: "FOO_TOKEN"

bar:
    - name: "bar"
    - environment-url: "https://bar.dynatrace-managed.com/e/environmentid"
    - env-token-name: "BAR_TOKEN"
```

Environments can also be grouped. Only one group per environment is allowed. Assign environments to groups with `group.environment:`
```
production.foo:
    - name: "foo"
    - environment-url: "https://foo.dynatrace.com"
    - env-token-name: "FOO_TOKEN"

production.bar:
    - name: "bar"
    - environment-url: "https://bar.dynatrace-managed.com/e/id"
    - env-token-name: "BAR_TOKEN"

```
## Configuration Structure

### Projects

Configuration files are ordered by project in the `projects` folder. Project folder can only contain:
- configurations
- or another project(s)

This means, it is possible to group projects into folders.

Combining projects and configurations in same folder is not supported.

There is no restriction in the depth of projects tree.

To get an idea, what are the possible combinations take a look at `cmd/monaco/test-resources/integration-multi-project`

### Config JSON Templates

The `json` files that can be uploaded with this tool are the jsons object that the respective Dynatrace APIs accept/return.

Adding a new config is generally done via the Dynatrace UI - unless you know the config JSON structures well enough to prefer writing them.

Configs can then be downloaded via the respective GET endpoint defined in the Dynatrace Configuration API, and should be cleaned up for auto-deployment.

Checked in configuration should not include:

* the entity's `id` but only it's `name`. The entity may be created or updated if one of the same name exists.
  * The `name` must be defined as [a variable](#configuration-yaml-structure).
* hardcoded values for environment information such as references to other auto-deployed entities, tags, management-zones, etc.
  * These should all be referenced as variables as [described below](#referencing-other-configurations).
* Empty/null values that are optional to when creating an object.
  * Most API GET endpoints return more data than needed to create an object. Many of those fields are empty or null, and can just be omited.
  * e.g. `tileFilter`s on dashboards

The tool handles these files as templates, so you can use variables in the format

```
{{ .variable }}
```

inside the config json.

Variables present in the template need to be defined in the respective config `yaml` - [see 'Configuration YAML Structure'](#configuration-yaml-structure).

#### Things you should know

##### Dashboard JSON

When you create a dashboard in the Dynatrace UI it will be private by default. All the dashboards deployed for **monaco** need to be shared publicly with other users.

You can change that in the dashboard settings, or by just changing the `json` you will check in.

A generally recommended value for the `dashboardMetadata` field is:

```
 "dashboardMetadata": {
    "name": "{{ .name }}",
    "shared": true,
    "sharingDetails": {
      "linkShared": true,
      "published": true
    },
    "dashboardFilter": {
      "timeframe": "",
      "managementZone": {
        "id": "{{ .managementZoneId }}",
        "name": "{{ .managementZoneName }}"
      }
    }
  }
```

This config does the following:
* References the name of the Dashboard as a [variable](#configuration-yaml-structure)
* Shares the dashboard with other users
* Sets a management zone filter on the complete dashboard, again as a variable, most likely [referenced from another config](#referencing-other-configurations)
  * Filtering the whole dashboard by management zone, makes sure no data not meant to be shown is accidentally picked up on tiles, and removes the possible need to define filters for individual tiles

##### Calculated log metrics JSON

When you create custom log metrics, you need to reference the metricKey of the log metric as `{{ .name }}` in the YAML file.

e.g.
```
{
  "metricKey": "{{ .name }}",
  "active": true,
  "displayName": "{{ .displayName }}",
  ...
}
```

##### Conditional naming JSON

As there is no `name` parameter in conditional naming API you should map `{{ .name }}` to `displayName`.

e.g.
```
{
  "type": "PROCESS_GROUP",
  "nameFormat": "Test naming PG for {Host:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

This also applies to the `HOST` type. eg.
```
{
  "type": "HOST",
  "nameFormat": "Test - {Host:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

Also applies to the `SERVICE` type. eg.
```
{
  "type": "SERVICE",
  "nameFormat": "{ProcessGroup:KubernetesNamespace} - {Service:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```
### Configuration Types / APIs

Each such type folder must contain one `configuration yaml` and one or more `json` files containing the actual configuration send to the Dynatrace API.

e.g.
```
projects/
        {projectname}/
                     {configuration type}/
                                         config.yaml
                                         config1.json
                                         config2.json
```

#### Supported Configuration Types

Supported configurations types are:
```
alerting-profile: /api/config/v1/alertingProfiles
management-zone: /api/config/v1/managementZones
auto-tag: /api/config/v1/autoTags
dashboard: /api/config/v1/dashboards
notification: /api/config/v1/notifications
extension: /api/config/v1/extensions
custom-service-java: /api/config/v1/service/customServices/java
anomaly-detection-metrics: /api/config/v1/anomalyDetection/metricEvents
synthetic-location: /api/v1/synthetic/locations
synthetic-monitor: /api/v1/synthetic/monitors
application: /api/config/v1/applications/web
app-detection-rule: /api/config/v1/applicationDetectionRules
aws-credentials: /api/config/v1/aws/credentials
kubernetes-credentials: /api/config/v1/kubernetes/credentials
azure-credentials: /api/config/v1/azure/credentials
request-attributes: /api/config/v1/service/requestAttributes
calculated-metrics-service: /api/config/v1/calculatedMetrics/service
calculated-metrics-log: /api/config/v1/calculatedMetrics/log
conditional-naming-processgroup: /api/config/v1/conditionalNaming/processGroup,
conditional-naming-host: /api/config/v1/conditionalNaming/host,
conditional-naming-service: /api/config/v1/conditionalNaming/service,
maintenance-window: /api/config/v1/maintenanceWindows
```

### Configuration YAML Structure

Every configuration needs a YAML containing required and optional content.

A minimal viable config needs to look like this:

```
config:
    - {config name} : "{path of config json template}"

{config name}:
    - name: "{a unique name}"
```

e.g. in `projects/infrastructure/alerting-profile/profiles.yaml`
```
config:
  - profile: "projects/infrastructure/alerting-profile/profile.json"

profile:
  - name: "profile-name"
[...]
```

Every config needs to provide a name for unique identification, omitting the name variable or using a duplicate name will result in a validation / deployment error.

Any defined `{config name}` represents a variable that can then be used in a [JSON template](#config-json-templates), and will be resolved and inserted into the config before deployment to Dynatrace.

e.g. `projects/infrastructure/alerting-profile/profiles.yaml` defines a `name`:
```
[...]
profile:
  - name: "EXAMPLE Infrastructure"
[...]
```

Which is then used in `projects/infrastructure/alerting-profile/profile.json` as `{{.name}}`.

### Skip configuration deployment

To skip configuration from deploying you can use predefined `skipDeployment` parameter. You can skip deployment of the whole configuration:

```
my-config:
  - name: "My config"
  - skipDeployment: "true"
```
enable it by default, but skip for environment or group:
```
my-config:
  - name: "My config"
  - skipDeployment: "true"

my-config.development:
  - skipDeployment: "false"
```
or disable it by default and enable only for environment or group:
```
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

```
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
```
  - managementZoneId: "projects/infrastructure/management-zone/zone.id"
```

### Referencing other json templates
Json templates are usually defined inside of project configuration and then references in same project:

**testproject/auto-tag/auto-tag.yaml:**
```
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
```
config:
  - application-tagging-multiproject: "/path/to/project/auto-tag/application-tagging.json"

application-tagging-multiproject:
  - name: "Test Application Multiproject"
```
This would save us of content duplication and redefining same templates over and over again.

Of course, it is also possible to reuse one template multiple times within one or different yaml file(s):
**testproject/auto-tag/auto-tag.yaml:**
```
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

```
development:
    - name: "Dev"
    - environment-url: "{{ .Env.DEV_URL }}"
    - env-token-name: "DEV"
```

To resolve an environment variable directly in the `json` is also possible. See the following example which sets the value
of an alerting profile from the env var `ALERTING_PROFILE_VALUE`.

```
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

### Plugin Configuration

> **Important**
>
> If you define something that depends on a metric created by a plugin, make sure to reference the plugin by name, so that the configurations will be applied in the correct order (after the plugin was created)
>
> Plugins can not be referenced by `id` as the Dynatrace plugin endpoint does not return this!
>
> Use only the plugin `name`

e.g. `projects/example-project/anomaly-detection-metrics/numberOfDistributionInProgressAlert.json` depends on the plugin defined by `projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.json`

So `projects/example-project/anomaly-detection-metrics/example-anomaly.yaml` references the plugin by name in a variable:

```
- metricPrefix: "projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.name"
```

to then construct the `metric-id` in the `json` as:

```
"metricId": "ext:{{.metricPrefix}}.metric_NumberOfDistributionInProgressRequests"
```

### Delete Configuration
Configuration which is not needed anymore can also be deleted in automated fashion. This tool is looking for `delete.yaml` file located in projects root
folder and deletes all configurations defined in this file after finishing deployment. `delete.yaml` file structure should be defined as following, where
beside from API you also have to specify then `name` (not id) of configuration to be deleted:
```
config:
  - "auto-tag/my-tag"
  - "custom-service-java/my custom service"
...
```

Warning: if the same name is used for the new config and config defined in delete.yaml, then config will be deleted right after deployment.