---
sidebar_position: 6
---

# Delete Configuration

Configuration which is not needed anymore can also be deleted in automated fashion. This tool is looking for `delete.yaml` file located in projects root
folder and deletes all configurations defined in this file after finishing deployment. `delete.yaml` file structure should be defined as following, where
beside from API you also have to specify then `name` (not id) of configuration to be deleted:
```yaml
delete:
  - "auto-tag/my-tag"
  - "custom-service-java/my custom service"
...
```

Warning: if the same name is used for the new config and config defined in delete.yaml, then config will be deleted right after deployment.