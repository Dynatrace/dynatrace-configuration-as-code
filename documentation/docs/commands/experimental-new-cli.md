---
sidebar_position: 3
---

# Experimental New CLI

With Monaco version 1.2.0+ an experimental CLI is available.  

To activate the new experimental CLI simply set an the env variable NEW_CLI to 1:

```shell title="shell"

 NEW_CLI=1 monaco

```

Instead of being flag based, the new CLI is based around commands.
As of right now the following commands are available:

- deploy
- download


### Deploy

This command is basically doing what the old tool did. It is used to deploy a specified config to a dynatrace environment. The flags to things like the environments files are mostly the same. Read more about it here: [Deploy projects](../commands/deploying-projects.md)

### Download

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. Read more about it here: [Download configuration](../commands/downloading-configuration.md)