---
sidebar_position: 5
---

# Logging

Sometimes it is useful for debugging to see HTTP traffic between Monaco and the Dynatrace API. This is possible by specifying a log file via the `MONACO_REQUEST_LOG` and `MONACO_RESPONSE_LOG` env variables.

The specified file can be relative or absolute. If relative, then it will be located relative to the current working dir.

> :warning: If the file already exists, it will get **truncated!**

To specify the log file, set the environment variable and Monaco will start writing all send requests to the file as follows:

```
 MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project
```

The content of multipart post requests is currently not logged. This is a known limitation.