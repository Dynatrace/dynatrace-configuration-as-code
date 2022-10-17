---
sidebar_position: 4
---

# Download

This feature lets you download the configuration from a Dynatrace tenant as Monaco files. 
Use this feature to avoid starting from scratch when using Monaco. 

## Download configurations

### Using the manifest

1. [Create a manifest file if you don't have one already](../configuration/yaml_configuration.md).
2. Run monaco using the download command

```shell
monaco download manifest manifest.yaml environment-name
```

Use `--help` to view all options you have to configure the download-behavior:
```shell
monaco download manifest --help`
```

### Direct download

To download an environment directly without the usage of a manifest, use the `monaco download direct`-command.
This command can get you started if you have nothing configured yet. A manifest will be created for you.

```shell
monaco download  direct https://environment.dynatracelabs.com API_TOKEN_ENVIRONMENT_VARIABLE  
```
The content of the environment variable is the api-token used to download the configuration.

Use `--help` to view all options you have to configure the download-behavior:
```shell
monaco download direct --help`
```


## Unsupported APIs

Some APIs are supported to be deployed but are not supported to be downloaded.
To deploy them, you need to create them manually. 

These APIs are:
* aws-credentials
* azure-credentials
* credential-vault
* extension
* kubernetes-credentials
