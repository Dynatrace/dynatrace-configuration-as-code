---
sidebar_position: 4
---

# Downloading Configuration

This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. For this feature you will have to enable CLI version 2.0.


### Steps

1. Enable CLI version 2.0 by adding an environment variable called NEW_CLI with a non-empty value other than 0. 
```shell title="shell"

$ export NEW_CLI=1

```
2. Create an environment file.
3. Run monaco using the download command i.e.

```shell title="shell"

$ monaco download --environments=my-environment.yaml

```

### Options

Instead of downloading all the configurations for all the APIs you can pass a list of API values separated by comma using the following flag `--downloadSpecificAPI`.

```shell title="shell"

$ monaco download --downloadSpecificAPI alerting-profiles,dashboard --environments=my-environment.yaml

```

### Notes

You should take into consideration the following limitations of the current process.

#### Application Detection Rules:

When using download functionality you will only be able to update existing application detection rules. If you want to create a new app detection rule you can only do so if there are no other app detection rules for that application.
