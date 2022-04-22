---
sidebar_position: 2
title: Install Monaco
---

import Tabs from "@theme/Tabs";
import TabItem from "@theme/TabItem";

This guide shows you how to download Monaco and install it on your operating system (Linux/macOS or Windows).

1.	Go to the Monaco [release page](https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases).
2.	Download the appropriate version.
3.	Check that the Monaco binary is available on your PATH. This process will differ depending on your operating system (see steps below). 

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

For Linux/macOS, we recommend using `curl`. You can download it from [here](https://curl.se/) or use `wget`.

```shell
# Linux
# x64
 curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-linux-amd64 -o monaco

# x86
 curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-linux-386 -o monaco

# macOS
 curl -L https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/releases/download/v1.7.0/monaco-darwin-10.16-amd64 -o monaco
```

Make the binary executable:

```shell
 chmod +x monaco
```

Optionally, install Monaco to a central location in your `PATH`.
This command assumes that the binary is currently in your downloads folder and that your $PATH includes `/usr/local/bin`:

```shell
# use any path that suits you; this is just a standard example. Install sudo if needed.
 sudo mv ~/Downloads/monaco /usr/local/bin/
```

Now you can execute the `monaco` command to verify the download. 

```shell
 monaco
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
   1.7.0
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
```

  </TabItem>
  <TabItem value="windows">

Before you start, you need to set the PATH on Windows: 

1.	Go to Control Panel -> System -> System settings -> Environment Variables.
2.	Scroll down in system variables until you find PATH.
3.	Click edit and change accordingly.
4.	Include a semicolon at the end of the previous as that is the delimiter, i.e., c:\path;c:\path2
5.	Launch a new console for the settings to take effect.

Once your PATH is set, verify the installation by running `monaco` from your terminal. 

```shell
 monaco
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
   1.7.0

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
```

  </TabItem>
</Tabs>
  </TabItem>
</Tabs>

Now that Monaco is installed, follow our introductory guide on [how to deploy a configuration to Dynatrace.](../configuration/deploy_configuration)
