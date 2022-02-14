---
sidebar_position: 1
---

# Validate configuration

Monaco validates configuration files in a directory by performing a dry run. It will check whether your Dynatrace config files are in a valid JSON format, and whether your tool configuration YAML files can be parsed and used.

To validate the configuration, execute a `monaco dry run` on a YAML file as shown below.

Create the file at `src/pages/my-react-page.js`:

```jsx title="run monaco in dry mode"
 ./monaco -dry-run --environments=project/sub-project/my-environments.yaml
2020/06/16 16:22:30 monaco v1.0.0
2020/06/16 16:22:30 Reading projects...
2020/06/16 16:22:30 Sorting projects...
...
2020/06/16 16:22:30 Config validation SUCCESSFUL
```