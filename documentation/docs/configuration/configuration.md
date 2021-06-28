---
sidebar_position: 1
---

# Configuration

The Dynatrace MONitoring As Code tool defines a structure on what files
it needs, to deploy something to an dynatrace environment.

There are multiple levels of configuration, which are:
- Deployment manifest
- Project

## Deployment manifest
This file defines the what and where. For more information see [here](./manifest.md)

## Projects
Are used to logically group api configurations together. An example of
a project could be e.g. a service. So all configuration regarding this
service will be present in the folder.

For more information see [here](./projects.md)

## Example

Please have a look at the example folder on the github project page.
