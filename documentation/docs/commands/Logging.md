---
sidebar_position: 5
---

# Logging

Sometimes it is useful for debugging to see http traffic between monaco and the dynatrace api. This is possible by specifying a log file via the `MONACO_REQUEST_LOG` and `MONACO_RESPONSE_LOG` env variables.

The specified file can either be relative, then it will be located relative form the current working dir, or absolute.

**NOTE**: If the file already exists, it will get **truncated!**

Simply set the environment variable and monaco will start writing all send requests to the file like:

```shell title="shell"

$ MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project

```

As of right now, the content of multipart post requests is not logged. This is a known limitation.
