---
sidebar_position: 2
title: Install Monaco
---

import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";

To use monaco you will need to install it. Monaco is distributed as a binary package.

To install Monaco, find the appropriate executable for your system and download it.

Ensure that the monaco binary is available on your PATH. This process will differ depending on your operating system. This process will differ depending on your operating system.

Executables are available in the [release page](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases).

<Tabs
  defaultValue="operating system"
  values={[
    { label: "Operating System", value: "operating system" }
  ]}
>
  <TabItem value="operating system">

<Tabs
  defaultValue="linux-macos"
  values={[
    { label: "Linux / macOS", value: "linux-macos" },
    { label: "Windows", value: "windows" },
  ]}
>
  <TabItem value="linux-macos">

This is an example using `curl`. If you don't have `curl`, install it, or use `wget`.

```shell
# Linux
# x64
$ curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.5.3/monaco-linux-amd64 -o monaco

# x86
$ curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.5.3/monaco-linux-386 -o monaco

# macOS
$ curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.5.3/monaco-darwin-10.12-amd64 -o monaco
```

Make the binary executable:

```shell
$ chmod +x monaco
```

Optionally install monaco to a central location in your `PATH`.
This command assumes that the binary is currently in your downloads folder and that your PATH includes `/usr/local/bin`:

```shell
# use any path that suits you, this is just a standard example. Install sudo if needed.
$ sudo mv ~/Downloads/monaco /usr/local/bin/
```

## Verify Download

```shell
$ monaco
You are currently using the old CLI structure which will be used by
default until monaco version 2.0.0
Check out the beta of the new CLI by adding the environment variable
  "NEW_CLI".
We can't wait for your feedback.
NAME:
   monaco-linux-amd64 - Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.
USAGE:
   monaco-linux-amd64 [global options] command [command options] [working directory]
VERSION:
   1.5.1
DESCRIPTION:
   Tool used to deploy dynatrace configurations via the cli
   Examples:
     Deploy a specific project inside a root config folder:
       monaco -p='project-folder' -e='environments.yaml' projects-root-folder
     Deploy a specific project to a specific tenant:
       monaco --environments environments.yaml --specific-environment dev --project myProject
COMMANDS:
   help, h  Shows a list of commands or help for one command
GLOBAL OPTIONS:
   --verbose, -v                             (default: false)
   --environments value, -e value            Yaml file containing environments to deploy to
   --specific-environment value, --se value  Specific environment (from list) to deploy to (default: none)
   --project value, -p value                 Project configuration to deploy (also deploys any dependent configurations) (default: none)
   --dry-run, -d                             Switches to just validation instead of actual deployment (default: false)
   --continue-on-error, -c                   Proceed deployment even if config upload fails (default: false)
   --help, -h                                show help (default: false)
   --version                                 print the version (default: false)
2021-05-04 11:25:04 ERROR Required flag "environments" not set
```

  </TabItem>
  <TabItem value="windows">

From the user interface, use this [Stack OverFlow](https://stackoverflow.com/questions/1618280/where-can-i-set-path-to-make-exe-on-windows) instructions to set the PATH on Windows.
Verify the installation by running `monaco`  from your terminal.

```shell
$ monaco
YYou are currently using the old CLI structure which will be used by
default until monaco version 2.0.0

Check out the beta of the new CLI by adding the environment variable
  "NEW_CLI".

We can't wait for your feedback.

NAME:
   monaco.exe - Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.

USAGE:
   monaco.exe [global options] command [command options] [working directory]

VERSION:
   1.5.0

DESCRIPTION:
   Tool used to deploy dynatrace configurations via the cli

   Examples:
     Deploy a specific project inside a root config folder:
       monaco -p='project-folder' -e='environments.yaml' projects-root-folder

     Deploy a specific project to a specific tenant:
       monaco --environments environments.yaml --specific-environment dev --project myProject

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --verbose, -v                             (default: false)
   --environments value, -e value            Yaml file containing environments to deploy to
   --specific-environment value, --se value  Specific environment (from list) to deploy to (default: none)
   --project value, -p value                 Project configuration to deploy (also deploys any dependent configurations) (default: none)
   --dry-run, -d                             Switches to just validation instead of actual deployment (default: false)
   --continue-on-error, -c                   Proceed deployment even if config upload fails (default: false)
   --help, -h                                show help (default: false)
   --version                                 print the version (default: false)
2021-05-06 14:19:32 ERROR Required flag "environments" not set
```

  </TabItem>
</Tabs>
  </TabItem>
</Tabs>
