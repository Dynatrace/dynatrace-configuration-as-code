---
sidebar_position: 4
---

# Download configuration

This feature lets you download the configuration from a Dynatrace tenant as Monaco files. 
Use this feature to avoid starting from scratch when using Monaco. 

> :warning: This feature requires CLI version 2.0.

## Download configurations


1. Enable CLI 2.0 by adding an environment variable called NEW_CLI with a non-empty value other than 0. 
```shell
export NEW_CLI=1
```
2. Create an environment file.
3. Run monaco using the download command

```shell
monaco download --environments=my-environment.yaml
```

## Options

To download specific APIs only, use `--specific-api` to pass an API. 


```shell
monaco download --specific-api alerting-profile --specific-api dashboard --environments=my-environment.yaml
```

# Skipped configurations

Some configurations on Dynatrace are predefined and must not be downloaded and reuploaded on the same, or another environment.

Those skipped configurations are:

| API                         | Details                                                                           |
|-----------------------------|-----------------------------------------------------------------------------------|
| `dashboard`/`dashboard-v2`  | Dashboards that have the owner `Dynatrace` set, or dashboards, which are a preset |
| `anomaly-detection-metrics` | Configurations that start with `dynatrace.` or `ruxit.`                           |
| `synthetic-locations`       | `public` (Dynatrace provided) locations                                           |
| `extension`                 | Extensions with an id that starts with `custom.`                                  |

## Notes

> :warning: **Application Detection Rules.** When using download functionality, you can only update existing application detection rules. You can only create a new app detection rule if no other app detection rules exist for that application.
