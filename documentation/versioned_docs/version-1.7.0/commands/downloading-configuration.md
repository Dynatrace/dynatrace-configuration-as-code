---
sidebar_position: 4
---

# Download configuration

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. 

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

> :warning: **Application Detection Rules.** When using download functionality you will only be able to update existing application dectection rules. If you want to create a new app detection rule you can only do so if there are no other app detection rules for that application.