# Dynatrace Monitoring as Code

This tool automates deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

**For release notes please see [RELEASE_NOTES.md](./RELEASE_NOTES.md)**

**If you wish to contribute please read [CONTRIBUTING.md](./CONTRIBUTING.md)**

**Table of Contents**
> Probably you're most interested in [Using Monitoring as Code (monaco) Tool](#using-monitoring-as-code-tool)
and [Configuration Structure](#configuration-structure)

- [Dynatrace Monitoring as Code](#dynatrace-monitoring-as-code)
    - [Install Monaco](#install-monaco)
      - [On Mac or Linux systems perform the following](#on-mac-or-linux-systems-perform-the-following)
      - [On Windows](#on-windows)
    - [Commands (CLI)](#commands-cli)
      - [Dry Run (Validating Configuration)](#dry-run-validating-configuration)
      - [Experimental new CLI](#experimental-new-cli)
        - [Deploy](#deploy)
        - [Download](#download)
      - [Misc](#misc)
        - [Logging all requests send to dynatrace](#logging-all-requests-send-to-dynatrace)
    - [Deploying Configuration to Dynatrace](#deploying-configuration-to-dynatrace)
      - [Running The Tool](#running-the-tool)
      - [Running The Tool With A Proxy](#running-the-tool-with-a-proxy)
      - [Environments file](#environments-file)
  - [Configuration Structure](#configuration-structure)
    - [Projects](#projects)
    - [Config JSON Templates](#config-json-templates)
      - [Things you should know](#things-you-should-know)
        - [Dashboard JSON](#dashboard-json)
        - [Calculated log metrics JSON](#calculated-log-metrics-json)
        - [Conditional naming JSON](#conditional-naming-json)
    - [Configuration Types / APIs](#configuration-types--apis)
      - [Supported Configuration Types and Token Permissions](#supported-configuration-types-and-token-permissions)
    - [Configuration YAML Structure](#configuration-yaml-structure)
    - [Skip configuration deployment](#skip-configuration-deployment)
    - [Specific Configuration per Environment or group](#specific-configuration-per-environment-or-group)
    - [Referencing other Configurations](#referencing-other-configurations)
    - [Referencing other json templates](#referencing-other-json-templates)
    - [Templating of Environment Variables](#templating-of-environment-variables)
    - [Plugin Configuration](#plugin-configuration)
    - [Delete Configuration](#delete-configuration)

---

### Install Monaco
To use monaco you will need to install it. Monaco is distributed as a binary package.

To install Monaco, find the appropriate [package](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/latest) for your system and download it as a zip archive.

After downloading Monaco, unzip the package. Monaco runs as a single binary named monaco.

Ensure that the monaco binary is available on your PATH. This process will differ depending on your operating system.

#### On Mac or Linux systems perform the following

Print a colon-separated list of locations in your ```PATH```:

```
$ echo $PATH
```

Move the Monaco binary to one of the listed locations. This command assumes that the binary is currently in your downloads folder and that your PATH includes ```/usr/local/bin```:

```
mv ~/Downloads/monaco /usr/local/bin/
```

#### On Windows

From the user interface, use [this Stack OverFlow](https://stackoverflow.com/questions/1618280/where-can-i-set-path-to-make-exe-on-windows) instructions to set the PATH on Windows.

Verify the installation by running ```monaco``` from your terminal.


### Commands (CLI)

Monitoring as Code is controlled via command-line interface.

The tool is a single command line application that takes required and optional arguments via flags such as `--environments`, `--project` or `--dry-run`.

The tool always depends on a config folder where all configuration projects are stored, possibly with further project subfolders.
If nothing is supplied the current working dir is used.

Running monaco is done with required and non-required options and positional arguments:

```
monaco --environments <path-to-environment-yaml-file> [--specific-environment <environment-name>] [--project <project-folder>] [--dry-run] [--verbose] [--continue-on-error] [projects-root-folder]
```

For deploying a specific project inside a root config folder, the tool could be run as:

```monaco --project <project-folder> --environments <path-to-environment-yaml-file> [projects-root-folder]```

In this case the **project** is within the **projects-root-folder**.

> Note that `[projects-root-folder]` needs to be a relative path from the directory you run monaco in.

For validating your complete configuration in the current folder, the tool could be run as:
```monaco --dry-run --environments <path-to-environment-yaml-file>```

For deploying all configurations to a single environment and get verbose output, the tool could be run as:
```monaco -v -e <path-to-environment-yaml-file> -se <name of environment>```

If, during deployment, `monaco` detects an error (configuration upload fails), it automatically stops deployment of affected environment. In case you want 
`monaco` to ignore errors and try to upload other configurations, you can provide `--continue-on-error` flag:
```monaco deploy --project <project-folder> --environments <path-to-environment-yaml-file> continue-on-error [projects-root-folder]```

Multiple projects can be specified as well:

```-p="project1,project2,project3"```

In order to get the version of the binary simply execute: 
```sh
monaco --version
```

*NOTE:* If the `--version` flag is present it will *ONLY* print the version and then exit. *ANY OTHER FLAG WILL BE IGNORED*.

The supported flags are described below:

```
   --verbose, -v                             (default: false)
   --environments value, -e value            Yaml file containing environments to deploy to
   --specific-environment value, --se value  Specific environment (from list) to deploy to (default: none)
   --project value, -p value                 Project configuration to deploy (also deploys any dependent configurations) (default: none)
   --dry-run, -d                             Switches to just validation instead of actual deployment (default: false)
   --continue-on-error, -c                   Proceed deployment even if config upload fails (default: false)
   --help, -h                                show help (default: false)
   --version                                 print the version (default: false)
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

#### Experimental new CLI
Starting with version 1.2.0 a new experimental CLI is available. The plan is that it 
will gradually become the new default in the next few releases. 

To activate the new experimental cli simply set an the env variable `NEW_CLI` to 1. 

E.g.

```sh
NEW_CLI=1 monaco 
```

By running the above example you will notice that instead of being flag based, the 
new cli is based around commands. 

As of right now the following commands are available:
* deploy
* download

##### Deploy
This command is basically doing what the old tool did. It is used to deploy a specified
config to a dynatrace environment. The flags to things like the environments files
are mostly the same. 

##### Download
This feature allows you to download the configuration from a Dynatrace
tenant as Monaco files. You can use this feature to avoid starting from
scratch when using Monaco. 

For more information on this feature, see [pkg/download/README.md](./pkg/download/README.md).

#### Misc
<a id="cli-misc"/>

##### Logging all requests/response send to dynatrace
<a id="cli-misc-log-requests">

Sometimes it is useful for debugging to see http traffic between monaco and the dynatrace api.
This is possible by specifying a log file via the `MONACO_REQUEST_LOG` and `MONACO_RESPONSE_LOG`
env variables.

The specified file can either be relative, then it will be located relative form the current 
working dir, or absolute. 

**NOTE:** If the file already exists, it will get **truncated**!

Simply set the environment variable and monaco will start writing all send requests to 
the file like:

```sh
$ MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project
```

As of right now, the content of multipart post requests is not logged. This is a known 
limitation. 

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

monaco -e=environments.yaml -se dev (deploy all projects in the current folder to the "dev" environment defined in environments.yaml)
```

If `project` contains additional sub-projects, then all projects are deployed recursively.

If `project` depends on different projects under the same root, those are also deployed.

Multiple projects could be specified by `-p="projectA, projectB, projectC/subproject"`

To deploy configuration the tool will need a valid API Token(s) for the given environments defined as `environment variables` - you can define the name of that env var in the environments file.

To deploy to 1 specific environment within a `environments.yaml` file, the `-specific-environment` or `-se` flag can be passed:

```bash
monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster
```

#### Running The Tool With A Proxy

In environments where access to Dynatrace API endpoints is only possible or allowed via a proxy server, monaco provides the options to specify the address of your proxy server when running a command:

```bash
HTTPS_PROXY=localhost:5000 monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster 
```

With the new CLI:

```bash
HTTPS_PROXY=localhost:5000 NEW_CLI=1 monaco deploy -e environments.yaml 
```


#### Environments file
environments are defined in the `environments.yaml` consisting of the environment url and the name of the environment variable to use for the API token.

Deployment could be done a single environment or several environments defined in the `environments.yaml` file.

A environment yaml file structure is of the form:

```yaml
foo:
    - name: "foo"
    - env-url: "https://foo.example.com"
    - env-token-name: "FOO_TOKEN_ENV_VAR"

bar:
    - name: "bar"
    - env-url: "https://bar.dynatrace-managed.com/e/environmentid"
    - env-token-name: "BAR_TOKEN_ENV_VAR"
```

Environments can also be grouped. Only one group per environment is allowed. Assign environments to groups with `group.environment:`
```yaml
production.foo:
    - name: "foo"
    - env-url: "https://foo.dynatrace.com"
    - env-token-name: "FOO_TOKEN_ENV_VAR"

production.bar:
    - name: "bar"
    - env-url: "https://bar.dynatrace-managed.com/e/id"
    - env-token-name: "BAR_TOKEN_ENV_VAR"

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

```json
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

From Dynatrace version 208 onwards, a dashboard configuration must:

- Have a property ownner, the property owner in dashboardMetadata is mandatory and must contain a not null value.
- The property sharingDetails in dashboardMetadata is not present anymore.

##### Calculated log metrics JSON

There is a know drawback to `monaco`'s workaround to the slightly off-standard API for Calculated Log Metrics, which needs you to follow specific naming conventions for your configuration: 

When you create custom log metrics, your configurations `name` needs to be the `metricKey` of the log metric. 

Additionally it is possible that configuration upload fails when a metric configuration is newly created and an additional configuration depends on the new log metric. To work around this, set both `metricKey` and `displayName` to the same value. 

You will thus need to reference at least the `metricKey` of the log metric as `{{ .name }}` in the JSON file (as seen below).

e.g. in the configuration YAML
```yaml
...
some-log-metric-config:
  - name: "cal.log:this-is-some-metric"
```

and in the corresponding JSON: 
```json
{
  "metricKey": "{{ .name }}",
  "active": true,
  "displayName": "{{ .name }}",
  ...
}
```

##### Conditional naming JSON

As there is no `name` parameter in conditional naming API you should map `{{ .name }}` to `displayName`.

e.g.
```json
{
  "type": "PROCESS_GROUP",
  "nameFormat": "Test naming PG for {Host:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

This also applies to the `HOST` type. eg.
```json
{
  "type": "HOST",
  "nameFormat": "Test - {Host:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

Also applies to the `SERVICE` type. eg.
```json
{
  "type": "SERVICE",
  "nameFormat": "{ProcessGroup:KubernetesNamespace} - {Service:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```
### Configuration Types / APIs

Each such type folder must contain one `configuration yaml` and one or more `json` files containing the actual configuration send to the Dynatrace API.
The folder name is case-sensitive and needs to be written exactly as in its definition in [Supported Configuration Types](#supported-configuration-types).

e.g.
```
projects/
        {projectname}/
                     {configuration type}/
                                         config.yaml
                                         config1.json
                                         config2.json
```

#### Supported Configuration Types and Token Permissions

These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint.

| Configuration                   | Endpoint                                        | Token Permission(s)                                                                                                 |
| ------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| alerting-profile                | _/api/config/v1/alertingProfiles_               | `Read Configuration` & `Write Configuration`                                                                        |
| anomaly-detection-metrics       | _/api/config/v1/anomalyDetection/metricEvents_  | `Read Configuration` & `Write Configuration`                                                                        |
| app-detection-rule              | _/api/config/v1/applicationDetectionRules_      | `Read Configuration` & `Write Configuration`                                                                        |
| application **deprecated in 2.0.0!**| _/api/config/v1/applications/web_           | `Read Configuration` & `Write Configuration`                                                                        |
| application-web **replaces application**| _/api/config/v1/applications/web_       | `Read Configuration` & `Write Configuration`                                                                        |
| application-mobile              | _/api/config/v1/applications/mobile_            | `Read Configuration` & `Write Configuration`                                                                        |
| auto-tag                        | _/api/config/v1/autoTags_                       | `Read Configuration` & `Write Configuration`                                                                        |
| aws-credentials                 | _/api/config/v1/aws/credentials_                | `Read Configuration` & `Write Configuration`                                                                        |
| azure-credentials               | _/api/config/v1/azure/credentials_              | `Read Configuration` & `Write Configuration`                                                                        |
| calculated-metrics-log          | _/api/config/v1/calculatedMetrics/log_          | `Read Configuration` & `Write Configuration`                                                                        |
| calculated-metrics-service      | _/api/config/v1/calculatedMetrics/service_      | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-host         | _/api/config/v1/conditionalNaming/host_         | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-processgroup | _/api/config/v1/conditionalNaming/processGroup_ | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-service      | _/api/config/v1/conditionalNaming/service_      | `Read Configuration` & `Write Configuration`                                                                        |
| credential-vault                | _/api/config/v1/credentials_                    | `Read Credential Vault Entries` & `Write Credential Vault Entries`                                                  |
| custom-service-java             | _/api/config/v1/service/customServices/java_    | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-dotnet           | _/api/config/v1/service/customServices/dotnet_  | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-go               | _/api/config/v1/service/customServices/go_      | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-nodejs           | _/api/config/v1/service/customServices/nodejs_  | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-php              | _/api/config/v1/service/customServices/php_     | `Read Configuration` & `Write Configuration`                                                                        |
| dashboard                       | _/api/config/v1/dashboards_                     | `Read Configuration` & `Write Configuration`                                                                        |
| extension                       | _/api/config/v1/extensions_                     | `Read Configuration` & `Write Configuration`                                                                        |
| kubernetes-credentials          | _/api/config/v1/kubernetes/credentials_         | `Read Configuration` & `Write Configuration`                                                                        |
| maintenance-window              | _/api/config/v1/maintenanceWindows_             | `Deprecated: Configure maintenance windows`                                                                         |
| management-zone                 | _/api/config/v1/managementZones_                | `Read Configuration` & `Write Configuration`                                                                        |
| notification                    | _/api/config/v1/notifications_                  | `Read Configuration` & `Write Configuration`                                                                        |
| request-attributes              | _/api/config/v1/service/requestAttributes_      | `Read Configuration` & `Capture request data`                                                                       |
| request-naming-service          | _/api/config/v1/service/requestNaming_          | `Read Configuration` & `Write Configuration`                                                                        |
| slo                             | _/api/v2/slo_                                   | `Read SLO` & `Write SLOs`                                                                                           |
| synthetic-location              | _/api/v1/synthetic/locations_                   | `Access problem and event feed, metrics, and topology` & `Create and read synthetic monitors, locations, and nodes` |
| synthetic-monitor               | _/api/v1/synthetic/monitors_                    | `Create and read synthetic monitors, locations, and nodes`                                                          |

For reference, refer to [this](https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication) page for a detailed
description to each token permission.

If your desired API is not in the table above, please consider adding it be following the instructions in 
[How to add new APIs](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/blob/main/docs/how-to-add-a-new-api.md).

### Configuration YAML Structure

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

```yaml
- metricPrefix: "projects/example-project/plugin/custom.jmx.EXAMPLE-PLUGIN-MY-METRIC.name"
```

to then construct the `metric-id` in the `json` as:

```json
"metricId": "ext:{{.metricPrefix}}.metric_NumberOfDistributionInProgressRequests"
```

### Delete Configuration
Configuration which is not needed anymore can also be deleted in automated fashion. This tool is looking for `delete.yaml` file located in projects root
folder and deletes all configurations defined in this file after finishing deployment. `delete.yaml` file structure should be defined as following, where
beside from API you also have to specify then `name` (not id) of configuration to be deleted:
```yaml
delete:
  - "auto-tag/my-tag"
  - "custom-service-java/my custom service"
...
```

Warning: if the same name is used for the new config and config defined in delete.yaml, then config will be deleted right after deployment.
