---
sidebar_position: 6
---

# Delete configuration

This guide shows you how to use the delete configuration tool to delete a configuration that is not needed.

The delete configuration tool looks for a `delete.yaml` file located in the project's root folder and deletes all configurations defined in this file after finishing deployment.
 
## File structure

The `delete.yaml` file structure should be as follows.  

```yaml
delete:
  - "auto-tag/my-tag"
  - "custom-service-java/my custom service"
...
```
You must specify the API and the `name` (not id) of the configuration to be deleted.

> :warning: if the same name is used for the new config and the config defined in delete.yaml, then the config will be deleted right after deployment.

> :warning: Due to the nature of single configuration endpoints (i.e. global oppossed to entity configuration) and non-uniquely named configurations (i.e. *dashboard* and *request-naming-service*) these configurations can not be deleted.
