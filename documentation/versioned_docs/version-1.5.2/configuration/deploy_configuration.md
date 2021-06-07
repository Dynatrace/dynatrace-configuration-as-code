---
sidebar_position: 1
---


# Deploying Configuration to Dynatrace

Monaco allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to a environment or a group of environments. This is done by passing the --project flag (or -p for short).


### Running The Tool

Below you find a few samples on how to run the tool to deploy your configurations:

```shell title="shell"
monaco -e=environments.yaml (deploy all projects in the current folder to all environments)

monaco -e=environments.yaml -p="project" projects-root-folder (deploy projects-root-folder/project and any projects in projects-root-folder it depends on to all environments)

monaco -e=environments.yaml -p="projectA, projectB" projects-root-folder (deploy projects-root-folder/projectA, projectB and dependencies to all environments)

monaco -e=environments.yaml -se dev (deploy all projects in the current folder to the "dev" environment defined in environments.yaml)
```

If `project` contains additional sub-projects, then all projects are deployed recursively.

If `project` depends on different projects under the same root, those are also deployed.

Multiple projects could be specified by `-p="projectA, projectB, projectC/subproject"`

To deploy configuration the tool will need a valid API Token(s) for the given environments defined as `environment variables` - you can define the name of that env var in the environments file.

To deploy to 1 specific environment within a `environments.yaml` file, the `-specific-environment` or -se flag can be passed:

```shell title="shell"
monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster
```

### Running The Tool With A Proxy

In environments where access to Dynatrace API endpoints is only possible or allowed via a proxy server, monaco provides the options to specify the address of your proxy server when running a command:

```shell title="shell"
HTTPS_PROXY=localhost:5000 monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster 
```

With the new CLI:

```shell title="shell"
HTTPS_PROXY=localhost:5000 NEW_CLI=1 monaco deploy -e environments.yaml 
```