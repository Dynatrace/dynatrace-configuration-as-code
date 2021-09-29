---
sidebar_position: 3
---

# Delete

The `monaco delete` command is a convenient way to remove one or more configurations from one or more Dynatrace tenants. Ideally, you will not want to delete long-lived configurations in your production environments, Monaco is sometimes used to manage ephemeral configurations in development environments, in which case you can easily use Monaco to clean up those temporary configurations.

Usage: `delete [command options] manifest.yaml delete.yaml`

The delete command takes two arguments as yaml files. A manifest file which contains the list of Dynatrace environments and a delete file where configurations defined are to be removed.

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

And a `delete.yaml` file with the given structure:

```yaml
delete:
  - "infrastructure/my custom service"
```

The following delete command will remove the `my custom service` configuration within the **infrastructure** directory from the development environment:

```shell
monaco delete deployment-file.yaml delete.yaml
```

```
Note: The delete file must be named delete.yaml
```

## Delete Options

The following options (flags) allow you to change various details about how the delete command executes and reports on the delete operation.

- `--environment` - (Optional) Use this option or `-e` to delete your configuration(s) from specific environments within your deployment manifest file.