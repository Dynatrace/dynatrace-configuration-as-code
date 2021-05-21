---
sidebar_position: 3
---

# Configuration Structure

Configuration files are ordered by `project` in the projects folder. Project folder can only contain:

- configurations
- or another project(s)

This means, it is possible to group projects into folders.

Combining projects and configurations in same folder is not supported.

There is no restriction in the depth of projects tree.

To get an idea, what are the possible combinations take a look at `cmd/monaco/test-resources/integration-multi-project`.

## Config JSON Templates

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