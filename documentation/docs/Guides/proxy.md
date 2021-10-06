---
sidebar_position: 2
---

## Running Monaco With A Proxy

In environments where access to Dynatrace API endpoints is only possible or allowed via a proxy server, monaco provides the options to specify the address of your proxy server when running a command:

```shell title="shell"
HTTPS_PROXY=localhost:5000 monaco deploy example.yaml 
```
