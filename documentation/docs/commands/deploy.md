---
sidebar_position: 1
---

# Deploy

The `monaco deploy` command executes the action of deploying configurations to environment(s) defined in a given [deployment manifest](./configuration/manifest.md) file.

Usage: `monaco deploy [command options] deployment-manifest`

The most straightforward in using  `monaco deploy` is to run it without any flags (command options) at all and by passing it a file name of your deployment manifest. By doing so, all configurations in the `project` section of the deployment manifest file are applied to all environments stated within the file.

## Example

Consider a [deployment manifest](./configuration/manifest.md) file called **deployment-file.yaml** with the given structure below:

```yaml
projects:
  - name: infrastructure
    path: infrastructure

environments:
  - group: development
    entries:
      - name: development
        url: "https://mytenant.live.dynatrace.com"
        token:
          name: "TestIt"
```
The following deploy command will apply the configuration(s) within the **infrastructure** directory to the development environment:

```shell
monaco deploy deployment-file.yaml
```

## Deploy Options

The following options (flags) allow you to change various details about how the deploy command executes and reports on the deploy operation.

- `--project` - Specifies the project(s) to be deployed. This option is used when you want to specify one or more projects to be applied to your environments. `-p` can be used as shorthand option.

- `--environment` - Use this option or `-e` to apply your configuration(s) to specific environments within your deployment manifest file.

- `--continue-on-error` - With this option, deployment is proceeded even if an error is encountered. Use this option to ensure configurations that are valid are applied to your environment(s). The shorthand option is `-c`.

- `--dry-run` - This option or `-d` validates configuration files and skips deployment. It will check whether your Dynatrace configuration files are valid JSON, and whether your tool configuration yaml files can be parsed and used.