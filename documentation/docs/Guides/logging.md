---
sidebar_position: 5
---

# Log Files

Use the `MONACO_REQUEST_LOG` and `MONACO_RESPONSE_LOG` environment variables to specify a file
that logs the HTTP traffic between Monaco and the Dynatrace API.
This is useful while working on Monaco's source code.

The path for the specified file can be absolute or relative to the current working directory.

> :warning: The specified file(s) **will be truncated!**.

To specify the log file, set the environment variables:

```shell title="Logging monaco requests and responses"
MONACO_REQUEST_LOG=request.log MONACO_RESPONSE_LOG=response.log monaco -e environment project
```

Monaco immediately starts writing all send requests to the specified file(s).

As of right now, the content of multipart post requests is not logged. This is a known limitation.
