---
sidebar_position: 1
title: Commands cheat sheet
---

# Commands cheat sheet

This commands cheat-sheet for Monaco describes the basic commands for managing your configuration files.

## Deploy command

| Command                                           | Description                                                                                                                                                                                       |
|---------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| <nobr>`deploy`</nobr>                             | Deploy configurations to the environment(s) defined in a given deployment manifest file.                                                                                                          |
| <nobr>`deploy --project` or `-p`</nobr>           | Specify one or more project(s) to be deployed to your environment(s).                                                                                                                             |
| <nobr>`deploy --environment` or `-e`</nobr>       | Apply your configuration(s) to specific environment(s) within your deployment manifest file.                                                                                                      |
| <nobr>`deploy --continue-on-error` or `-c`</nobr> | Proceed with deployment even if an error is encountered. Ensure configurations that are valid are applied to your environment(s).                                                                 |
| <nobr>`deploy --dry-run` or `-d`</nobr>           | Validate configuration files and skip deployment. It will check whether your Dynatrace configuration files are valid JSON, and whether your tool configuration yaml files can be parsed and used. |

## Download command

| Command                                                              | Description                                                                                                                 |
|----------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| <nobr>`download manifest manifest.yaml environment`</nobr>           | Download all configuration from the environment 'environment' specified in the manifest.yaml                                |
| <nobr>`download direct https://env.dynatracelabs.com TOKEN`</nobr>   | Download all configuration from the environment at 'https://env.dynatracelabs.com' using the environment variable 'TOKEN'   |
| <nobr>`download [manifest/direct] [args] --output-folder`</nobr>     | Where to store the downloaded project.                                                                                      |
| <nobr>`download [manifest/direct] [args] --project`</nobr>           | The name of the project containing all downloaded configs.                                                                  |
| <nobr>`download [manifest/direct] [args] --specific-api`</nobr>      | The name of all APIs that need to be downloaded.                                                                            |
| -------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |

## Delete command

| Command                                         | Description                                                                                   |
|-------------------------------------------------|-----------------------------------------------------------------------------------------------|
| <nobr>`delete manifest.yaml delete.yaml`</nobr> | Remove one or more configurations from one or more Dynatrace tenants.  **Arguments required** |
| <nobr>`delete --environment` or `-e`</nobr>     | Delete your configuration(s) from specific environments within your deployment manifest file. |

## Convert command

| Command                                            | Description                                                                                                                  |
|----------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------|
| <nobr>`convert environment.yaml v1-project`</nobr> | Apply automatic conversion rules to help convert Monaco v.1 configuration files to Monaco v.2 files. **Arguments required.** |
| <nobr>`convert --manifest or -m`</nobr>            | Specify the name to be used for the manifest file produced by the convert command.                                           |
| <nobr>`convert --output-folder` or `-o`</nobr>     | Specify the name of the output folder created by the convert command to store all converted configurations.                  |
