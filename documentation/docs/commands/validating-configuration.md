---
sidebar_position: 1
---

# Validating Configuration

Monaco validates the configuration files in a directory, it does so by performing a dry run. It will check whether your Dynatrace config files are valid JSON, and whether your tool configuration yaml files can be parsed and used.

To validate the configuration execute monaco -dry-run on a yaml file as show here:


Create a file at `src/pages/my-react-page.js`:

```jsx title="run monaco in dry mode"
$ ./monaco -dry-run --environments=project/sub-project/my-environments.yaml
2020/06/16 16:22:30 monaco v1.0.0
2020/06/16 16:22:30 Reading projects...
2020/06/16 16:22:30 Sorting projects...
...
2020/06/16 16:22:30 Config validation SUCCESSFUL
```
