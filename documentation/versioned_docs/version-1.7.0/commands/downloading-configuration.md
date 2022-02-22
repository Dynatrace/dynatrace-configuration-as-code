---
sidebar_position: 4
---

# Download configuration

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. Use this feature to avoid starting from scratch when using Monaco. 

> :warning: This feature requires CLI version 2.0.

## Download configurations


1. Enable CLI 2.0 by adding an environment variable called NEW_CLI with a non-empty value other than 0. 
```shell title="shell"

 export NEW_CLI=1

```
2. Create an environment file.
3. Run monaco using the download command

```shell title="shell"

 monaco download --environments=my-environment.yaml

```

## Options

To download specific APIs only, use `--downloadSpecificAPI` to pass a list of API values separated by a comma. 

```shell title="shell"

 monaco download --downloadSpecificAPI alerting-profile,dashboard --environments=my-environment.yaml

```

## Notes

> :warning: **Application Detection Rules.** When using download functionality, you can only update existing application dectection rules. You can only create a new app detection rule if no other app detection rules exist for that application.
