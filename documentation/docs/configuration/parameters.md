---
sidebar_position: 4
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
an array, nor a map, you can simply provide the value.

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

## Compound Parameter

The compound parameter is a parameter composed of other parameters of the same
config. This parameters requires 2 properties: a `format` string and a list of
`references` to all referenced parameters. Both properties are required.
The `format` string can be any string, and to use parameters in it, the
following syntax is used: `{{ .parameter }}`, where `<parameter>` is the
name of the parameter that will be filled in. A simple example might look like this:

```yaml
parameters:
  example:
    type: compound
    format: "{{ .greeting }} {{ .entity }}!"
    references:
      - greeting
      - entity
  greeting: "Hello"
  entity: "World"
```

This would produce the value `Hello World!` for `example`. Compound parameters
can also be used for more complex values, as seen in the following example:

```yaml
parameters:
  example:
    type: compound
    format: "{{ .resource.name }}: {{ .resource.percent }}%"
    references:
      - resource
  progress:
    type: value
    value:
      name: "Health"
      percent: 40
```

This would produce the value `Health: 40%` for example.
Even though referenced parameters can only be from the same config,
by using the reference parameter it is possible to make a compound
parameters with other configs. The same goes for environment variables.

```yaml
parameters:
  example:
    type: compound
    format: "{{ .user }}'s dashboard is {{ .status }}"
    references:
      - user
      - status
  user:
    type: environment
    name: USER_NAME
  status:
    type: reference
    api: dashboard
    config: dashboard
    property: status
```

## List Parameter

Parameters of type `list` allow you to define lists of Value Parameters.

When written into a Template, these will be written as a JSON list surrounded
by square-brackets and seperated by commas. 

This type of parameter is generally useful when you require a simple list of things like emails, identifiers, etc., 
but can be filled with any kind of Value parameter.

Example:

```yaml
parameters:
  recipients:
    type: list
    values: 
        - first.last@company.com
        - someone.else@company.com
  geolocations: 
    type: list
    values: ["GEOLOCATTION-1234567", "GEOLOCATION-7654321"]
```

In the example above you see that you can define the list values either line by line, 
or as an array in YAML.

When using a List Parameter Value in a JSON Template, make sure to just reference the value without any extra brackets. 
 
 ```json
{
    "emails": {{ .recipients }}
}
```
 
 > **Note** that this differs from the sometimes used string list in v1 for which the template needed to include square brackets
> (e.g. `"emails": [ {{ .recipients }} ]`).
> 
> When such lists are encountered upon converting v1 configuration Templates will be automatically updated.
