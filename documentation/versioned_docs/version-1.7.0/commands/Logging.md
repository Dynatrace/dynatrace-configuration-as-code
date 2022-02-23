---
sidebar_position: 5
---

# Logging

Use the `MONACO_REQUEST_LOG` and `MONACO_RESPONSE_LOG` environment variables to specify a file that logs the HTTP traffic between Monaco and the Dynatrace API.
This is useful when debugging your implementation.

The path for the specified file can be absolute or relative to the current working directory.

> :warning: If the file already exists, it will get **truncated!**.

To specify the log file, set the environment variables:

```
 MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project
```

Monaco immediately starts writing all send requests to the specified file(s).

The content of multipart post requests is currently not logged. This is a known limitation.
