
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

| Flag name              | Short |  Multi  | Default                             |     Global     | Components                        | Description                                                                     |
|------------------------|-------|:-------:|-------------------------------------|:--------------:|-----------------------------------|---------------------------------------------------------------------------------|
| --verbose              | -v    |    ✗    | `false`                             | ✗ (but should) | deploy, download, delete, convert | Use debug mode                                                                  |
| --help                 | -h    |    ✗    | N/A                                 |       ✓        |                                   | Print help                                                                      |
| --continue-on-error    | -c    |    ✗    | `false`                             |       ✗        | deploy                            | Proceed even if an error occurs                                                 |
| --dry-run              | -d    |    ✗    | `false`                             |       ✗        | deploy                            | Use validation mode                                                             |
| --environments         | -e    |    ✓    | `[ ]`                               |       ✗        | deploy<br/>delete                 | What environments to deploy                                                     |
| --project              | -p    | ✓<br/>✗ | `[ ]`<br/>*required flag*           |       ✗        | deploy<br/>download               | What projects to deploy<br/>In what project-folder to save the downloaded files |
| --manifest             | -m    |    ✗    | *required flag*<br/>`manifest.yaml` |       ✗        | download<br/>convert              | What manifest file to use                                                       |
| --specific-environment | -s    |    ✗    | *required flag*                     |       ✗        | download                          | What specific environment in the manifest to use                                |
| --url                  | -u    |    ✗    | *required flag*                     |       ✗        | download                          | The URL to use                                                                  |
| --token                | -t    |    ✗    | *required flag*                     |       ✗        | download                          | The environment variable to use to download                                     |
| --specific-api         | -a    |    ✓    | `[ ]`                               |       ✗        | download                          | The list of apis to download, if not specified all are used                     |
| --output-folder        | -o    |    ✗    | `{project-folder}-v2`               |       ✗        | convert                           | The directory to put the converted files                                        |        

Inconsistencies to get rid of:
1. Required flags. We should not have required flags.
2. --manifest has different default values
3. --project has different meanings
4. --verbose should be a global flag

Improvements visible based on the above table
1. --verbose should be a global flag
2. --continue-on-error should also be available on delete and download.
