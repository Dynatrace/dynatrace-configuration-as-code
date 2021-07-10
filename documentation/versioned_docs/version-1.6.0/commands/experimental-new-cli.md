---
sidebar_position: 3
---

# Experimental New CLI

Starting with version 1.2.0 a new experimental CLI is available. The plan is that it will gradually become the new default in the next few releases.
 To activate the new experimental cli simply set an the env variable NEW_CLI to 1:

```shell title="shell"

$ NEW_CLI=1 monaco

```

By running the above example you will notice that instead of being flag based, the new cli is based around commands.
As of right now the following commands are available:

- deploy
- download


### Deploy

This command is basically doing what the old tool did. It is used to deploy a specified config to a dynatrace environment. The flags to things like the environments files are mostly the same.

### Download

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco.

### Dry run 
To validate a configuration while using the new CLI version, use the `deploy` command with the flag, `--dry-run`. For example, 
```
$ ./monaco deploy --dry-run <any other arguments>
```
