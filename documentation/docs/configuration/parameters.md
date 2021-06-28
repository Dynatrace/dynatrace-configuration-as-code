---
sidebar_position: 5
---

# Configuration Parameters

Parameters are used to provide values in configuration templates.
They are defined as yaml objects with a `type` entry. This
`type` then further decides how the paremeter object is interpreted.

One important property of parameters is, that they are lazy. This
means a value is only then evaulated, if the parameter is actually
referenced by a configuration which is going to be deployed.

The following parameter types are currently available:
* Value
* Environment
* Reference

## Value Parameter

The value parameter is the simplest form of parameter. Besides the
`type` property, it offers also `value`. You can define whatever you
like as the value. Even nested maps. This value is then accessible in
the template file.

Since `values` are the most common type of parameter, there is also a
special short form syntax to define them. If your parameter is neither
an array, nor an map, you can simply provide the value.

Example:

```yaml
parameters:
  threshold: 15
  complexThreshold:
    type: value
    value:
        amount: 15
        unit: sec
```

In the template of this config you could then access the `threshold`
parameter via `{ .threshold }`. To access e.g. the `amount` of the
`complexThreshold` you could use `{ .complexThreshold.amount }`.

## Environment Parameter

Parameters of type `environment` allow you to reference a environment
variable. The name of the env variable to reference is defined via a
`name` property. It is also possible to provide a default value, should
the env variable not be present. This can be done via the `default`
property.

If the `default` property is not set and the env variable is missing,
the parameter cannot be resolved. This will fail the deployment.
**Note** this is only the case, if the paramter is relevant to being
deployed. Parameters not referenced by the config to deploy are not
evaluated.

Example:

```yaml
parameters:
  owner:
    type: environment
    name: OWNER
    default: "-"
  target:
    type: environment
    name: TARGET
```

In this example, the `owner` parameter will evaluate to whatever value the
`OWNER` env variable is set to. If the env variable is not present, it
will evaluate to value `-`.

The `target` parameter will evaluate to the value of the `TARGET` env variable.
It will fail the deployment, if the variable is not set at deploy time.

## Reference Parameter

Since it is often required to reference some form of property of another
configuration, monaco offers a special reference parameter. This parameter
allows one configuration to depend on pretty much any parameter of another
config. To archive this, one has to specify the `project`, `api`,`config` and
`property` properties, to tell monaco where it gets its value. There is
also a short notation, which is an array. The syntax for this short
notation is like `[ "{project name}", "{api}", "{config name}", "{property}"" ]`
(note that the values in `{}` have to be replaced with real values, like
`["project-1", "management-zone", "main", "id"]`). Property can be any
parameter of the target config, the `name` or the `id`. `id` in this case
is the one from the dynatrace api. Monaco will make sure, that the deployment
of configuration is ordered and that the dependant config is deployed first.

If you configure a cycle of dependencies, the deployment will fail.

It is also possible to leave fields like project empty. It will then get filled
with the value from the current config. **Note** that it is not allowed to leave
a gap. You can only leave the top most level empty. For example if you have a
dashboard configuration `test-dashboard` in the project `development` and a
management-zone config `main` in the same project, you can reference the `id`
property of the mangement zone as `["management-zone", "main", "id"]` from the
dashboard. This would be equivalent to

```yaml
type: reference
api: management-zone
config: main
property: id
```

Here a full example:
infrastructure/management-zone/config.yaml
```yaml
configs:
- id: main
  config:
    name: "Main zone"
    template: "zone.json"
```

development/management-zone/config.yaml
```yaml
configs:
- id: development
  config:
    name: "Development zone"
    template: "zone.json"
```

development/dashboard/config.yaml
```yaml
configs:
- id: overview
  config:
    name: "Overview dashboard"
    template: "dashboard.json"
    parameters:
      zoneId: ["infrastructure", "management-zone", "main", "id"]
      devZoneId: ["management-zone", "development", "id"]
```
