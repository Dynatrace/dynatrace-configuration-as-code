---
sidebar_position: 3
---

# Projects

A project is a folder containing, which contains specially named
sub-folders, representing APIs. This API folders then contain another
layer of folders defining confiugrations. Then finally, this configuration
folders contain yaml files, specifying what gets deployed.

## APIs

To see a list of all supported APIs and folder names, please have a
look [here](./configTypes_tokenPermissions.md).

## Configurations

Configurations consists of two parts:
- Yaml defining parameters, dependencies, name and template
- JSON Template file

### Configuration Yaml

Contains basic information about the config to deploy. This includes
the name of the config, the location of the template file and parameters
usable in the template file. Parameters can be overwritten based on what
group or environment is currently deployed.

For more details on the configuration syntax, see [here](yaml_config.md).

### JSON Template File

The JSON template contains the payload, which will get uploaded to the dynatrace
api endpoints. It allows you to reference all defined parameters of the configuration
via `{{ .[PARAMETER_NAME] }}` syntax.

Here is a basic example of how such a JSON might look like:
```json
{
    "name": "{{ .name }}",
    "type": "{{ .type }}"
}
```

And here the corresponding config yaml:
```yaml
configs:
- id: sample
  config:
    name: "Sample"
    parameters:
      type: "simple"
```

As you can see, it is also possible to reference the name of a configuration.

Under the hood monaco uses a technology called GO templates. In theory, they allow
you do define more complex templates, but it is **highly** recommended to keep templates
**as simple as possible**. This means that only knowing about referncing variables via
`{{ .[PARAMETER_NAME] }}` should be more than enough!

Here a [link](https://golang.org/pkg/text/template/) to the GO template documentation.

#### Things you should know

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
