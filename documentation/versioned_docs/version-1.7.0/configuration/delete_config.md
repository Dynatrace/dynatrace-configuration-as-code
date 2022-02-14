---
sidebar_position: 6
---

# Delete configuration

This guide will show you how to use the delete configuration tool to delete configuration that is not needed.

The delete configuration tool looks for a `delete.yaml` file located in the projects root folder and deletes all configurations defined in this file after finishing deployment.
 
## File structure

The `delete.yaml` file structure should be as follows.  

```yaml
delete:
  - "auto-tag/my-tag"
  - "custom-service-java/my custom service"
...
```
Beside from the API, you also have to specify the `name` (not id) of the configuration to be deleted.

> :warning: if the same name is used for the new config and the config defined in delete.yaml, then the config will be deleted right after deployment.