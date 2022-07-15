---
sidebar_position: 1
---

# Validating configuration

Monaco validates configuration files in a directory by performing a dry run. 
It will check whether your Dynatrace config files are in a valid JSON format and 
whether your tool configuration YAML files can be parsed and used.

To validate the configuration, execute a `monaco --dry-run` on a YAML file as shown below.

```shell title="Validating your configuration"
monaco --dry-run --environments=environments.yaml
2020/06/16 16:22:30 monaco v1.0.0
2020/06/16 16:22:30 Reading projects...
2020/06/16 16:22:30 Sorting projects...
...
2020/06/16 16:22:30 Config validation SUCCESSFUL
```

## Validating your configuration using the new CLI

To validate your configuration [using the new CLI](./experimental-new-cli.md) add the `--dry-run` flag to the `deploy` command.
```shell title="Validating your configuration using the new CLI"
NEW_CLI=1 monaco deploy --dry-run --environments=environments.yaml
```
