---
sidebar_position: 5
---

# Plugin configuration

> :warning: **Important**
>
> If you define something that depends on a metric created by a plugin, make sure to reference the plugin by name, so that the configurations will be applied in the correct order (after the plugin is created)
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

### Custom Extensions

Monaco can deploy custom extensions and handles the zipping of extensions; as such, the JSON file that defines an extension can just be checked in. The version of the extension is checked before it is uploaded. If the version of the extension to be uploaded is the same or equal to what is already deployed, then the upload is skipped.
An example of a custom extension can be found [here](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/tree/main/cmd/monaco/test-resources/integration-all-configs/project/extension).
