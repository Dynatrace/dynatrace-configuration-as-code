---
sidebar_position: 3
---

# Experimental New CLI

Monaco version 1.2.0+ includes the Beta version of the new CLI that is planned for a future release.
The new CLI is based on commands rather than flag based.
Currently, the following commands are available:

- deploy
- download

To activate the new experimental CLI, set the environment variable `NEW_CLI` to any non-empty value other than 0.

```shell title="Running monaco using the new CLI"
NEW_CLI=1 monaco
```

### Deploy

This command is basically doing what the old tool did. It is used to deploy a specified config to a Dynatrace environment.
The flags to things like the environments files are mostly the same.
Read more about it here: [Deploy projects](deploying-projects.md)

### Download

This command allows you to download the configuration from a Dynatrace tenant as Monaco files.
Use this command to avoid starting from scratch when using Monaco.
Read more about it here: [Download configuration](downloading-configuration.md)
