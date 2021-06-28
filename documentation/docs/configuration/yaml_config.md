---
sidebar_position: 4
---

# Configuration YAML

Each configuration yaml contains a list of configurations to be deployed.

A basic configuraion yaml looks something like this

```yaml
configs:
- id: test-dashboard
  config:
    name: Test Dashboard
    template: dashboard.json
    parameters:
      owner: Test User
```

As you can see the top level element is `configs`. Its value is a list of
configurations.

Each configuration requires a number of fields. The first field is `id`,
Then there is the config field. Those two fields are requried. it is also
possible to override values from config on a per group and environment
level. For this, there exists the `groupOverrides` and `environmentOverrides`
fields.

## ID

The `id` field is used to identify a config within the configurations. It
has to be unique for on an api level per project. So it is possible to have
e.g. two dashboards with the same `id` in two different projects.

It is important to note, that the field is only local to monaco. It has nothing
to do with the id provided by the dynatrace api. One important use case for this
`id` is, that it is used when using (reference parameters)[./parameters.md#reference_parameter].


## Config

The `config` field offers the following fields:
* `name` - **required** - Name used to identify objects in the dynatrace api
* `template` - **required** - Defines templating file used to render request to dynatrace api (see [here](./projects.md#template_file) for more details)
* `skipDeployment` - Boolean flag (either true, or false) used to notify monaco to not deploy this configuration
* `parameters` - List of parameters available in the template

### Config - Parameters

There are a number of different parameter types available.
Please see [here](./parameters.md) for more details.
