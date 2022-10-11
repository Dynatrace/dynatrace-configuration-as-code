
# Monaco CLI conventions

## Arguments for required arguments, Flags for non-required arguments

If an argument is required to make the command work, use it without flags. This also means, that the order of arguments is important.
If an argument is optional, make it a flag. With that, the order is also not important.

```shell
monaco deploy flagValue1 -a dashboards manifest.yaml
```

In the above sample, manifest.yaml is always required, so it is not using a flag.
On the opposite, what APIs to use is, so it is using a flag.

## Consistency

Here a list of all flags, so that they can be used consistently. Note that they are not consistent *yet*.

Note: The legacy deploy command is not included.

| Flag name              | Short |  Multi  | Default                                          | Global | Components           | Description                                                                     |
|------------------------|-------|:-------:|--------------------------------------------------|:------:|----------------------|---------------------------------------------------------------------------------|
| --verbose              | -v    |    ✗    | `false`                                          |   ✓    |                      | Enable debug logging                                                            |
| --help                 | -h    |    ✗    | N/A                                              |   ✓    |                      | Print help                                                                      |
| --continue-on-error    | -c    |    ✗    | `false`                                          |   ✗    | deploy               | Proceed even if an error occurs                                                 |
| --dry-run              | -d    |    ✗    | `false`                                          |   ✗    | deploy               | Use validation mode                                                             |
| --environments         | -e    |    ✓    | `[ ]`                                            |   ✗    | deploy<br/>delete    | What environments to deploy                                                     |
| --project              | -p    | ✓<br/>✗ | `[ ]`<br/>`project`                              |   ✗    | deploy<br/>download  | What projects to deploy<br/>In what project-folder to save the downloaded files |
| --manifest             | -m    |    ✗    | `manifest.yaml`                                  |   ✗    | convert              | What manifest file to use                                                       |
| --specific-api         | -a    |    ✓    | `[ ]`                                            |   ✗    | download             | The list of apis to download, if not specified all are used                     |
| --output-folder        | -o    |    ✗    | `{project-folder}-v2`<br/>`download-{timestamp}` |   ✗    | convert<br/>download | The directory to put the converted/downloaded files                             |        

Inconsistencies to get rid of:
1. `--project` has different meanings

Improvements visible based on the above table
1. `--continue-on-error` should also be available on delete and download.
