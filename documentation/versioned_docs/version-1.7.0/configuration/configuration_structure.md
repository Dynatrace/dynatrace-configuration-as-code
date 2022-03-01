---
sidebar_position: 3
---

# Configuration structure

Configuration files are ordered by `project` in the projects folder. Project folders can only contain:

- configurations
- other project(s)

This means it is possible to group projects into folders, but combining projects and configurations in the same folder is not supported.

There are no restrictions on the depth of a projects tree.

To get an idea of the possible combinations take a look at `cmd/monaco/test-resources/integration-multi-project`.

## Config JSON Templates

The `json` files that can be uploaded with this tool are the JSON objects that the respective Dynatrace APIs accept/return.

Adding a new config is generally done via the Dynatrace UI - unless you know the config JSON structures well enough to prefer writing them.

Configs can then be downloaded via the respective GET endpoint defined in the Dynatrace Configuration API, and should be cleaned up for auto-deployment.

Checked in configuration should **not** include:

* The entity's `id` but only its `name`. The entity may be created or updated if one of the same name exists.
  * The `name` must be defined as [a variable](#configuration-yaml-structure).
* Hardcoded values for environment information such as references to other auto-deployed entities, tags, management-zones, etc.
  * These should all be referenced as variables as [described below](#referencing-other-configurations).
* Empty/null values that are optional for the creation of an object.
  * Most API GET endpoints return more data than needed to create an object. Many of those fields are empty or null, and can be omitted.
  * E.g., `tileFilter`s on dashboards

The tool handles these files as templates, so you can use the following variable format inside the config JSON: 

```
{{ .variable }}
```


Variables present in the template need to be defined in the respective config `yaml` - [see 'Configuration YAML Structure'](../configuration/yaml_config).

### Dashboard JSON

When you create a dashboard in the Dynatrace UI it is private by default. All the dashboards deployed for Monaco need to be shared publicly with other users.

You can change this in the dashboard settings, or by just changing your checked in `json` file.

We recommend the following values for the `dashboardMetadata`:

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
* Reference the name of the Dashboard as a [variable](../configuration/yaml_config)
* Share the dashboard with other users
* Set a management zone filter on the complete dashboard, again as a variable, most likely [referenced from another config](../configuration/yaml_config#referencing-other-configurations)
  * Filtering the whole dashboard by management zone makes sure no private data is accidentally picked up on tiles, and removes the possible need to define filters for individual tiles

From Dynatrace version 208 onwards:

- A dashboard configuration must have a property owner. The property owner in dashboardMetadata is mandatory and must contain a non-null value.
- The property sharingDetails in dashboardMetadata is no longer present.

### Calculated log metrics JSON

There is a know drawback to Monaco's workaround to the slightly off-standard API for Calculated Log Metrics, which needs you to follow specific naming conventions for your configuration: 

> When you create custom log metrics, your configuration's `name` must be the `metricKey` of the log metric. 

Additionally it is possible that a configuration upload fails when a metric configuration is newly created and an additional configuration depends on the new log metric. To work around this, set both `metricKey` and `displayName` to the same value. 

You will thus need to reference at least the `metricKey` of the log metric as `{{ .name }}` in the JSON file, as you can see below: 

In the configuration YAML,

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

### Conditional naming JSON

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

This also applies to the `HOST` type.

```json
{
  "type": "HOST",
  "nameFormat": "Test - {Host:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

And it also applies to the `SERVICE` type. 

```json
{
  "type": "SERVICE",
  "nameFormat": "{ProcessGroup:KubernetesNamespace} - {Service:DetectedName}",
  "displayName": "{{ .name }}",
  ...
}
```

### Configuration types / APIs

Each type of folder must contain one `configuration yaml` and one or more JSON files containing the actual configuration sent to the Dynatrace API.
The folder name is case-sensitive and needs to be written exactly as in its definition in [Supported configuration types](../configuration/configTypes_tokenPermissions).


```
projects/
        {projectname}/
                     {configuration type}/
                                         config.yaml
                                         config1.json
                                         config2.json
```
