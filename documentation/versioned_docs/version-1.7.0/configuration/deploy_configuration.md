---
sidebar_position: 1
---


# Deploy configuration

This guide will show you how to deploy a Monaco configuration to Dynatrace. 

Monaco allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to an environment or a group of environments. This is done by passing the `--project` flag (or `-p` for short).


### Running the tool

Below you will find a few examples on how to run the tool to deploy your configurations:

```
monaco -e=environments.yaml (deploy all projects in the current folder to all environments)

monaco -e=environments.yaml -p="project" projects-root-folder (deploy projects-root-folder/project and any projects in projects-root-folder it depends on to all environments)

monaco -e=environments.yaml -p="projectA, projectB" projects-root-folder (deploy projects-root-folder/projectA, projectB and dependencies to all environments)

monaco -e=environments.yaml -se dev (deploy all projects in the current folder to the "dev" environment defined in environments.yaml)
```

If `project` contains additional sub-projects, then all projects are deployed recursively.

If `project` depends on different projects under the same root, those are also deployed.

Multiple projects can be specified with the following syntax: `-p="projectA, projectB, projectC/subproject"`

To deploy configurations the tool will need a valid API Token(s) for the given environments defined as `environment variables`. You can define the name of that enviroment variable in the environments file.

To deploy to one specific environment within an `environments.yaml` file, the `-specific-environment` or `-se` flag can be passed as follows:

```
monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster
```
Read more about the environments file here: [Environments file](./environments_file)

### Running the tool with a proxy

In environments where the access to Dynatrace API endpoints is only possible or allowed via a proxy server, Monaco provides the option of specifying the address of your proxy server when running a command:

```
HTTPS_PROXY=localhost:5000 monaco -e=environments.yaml -se=my-environment -p="my-environment" cluster 
```

With the new CLI:

```
HTTPS_PROXY=localhost:5000 NEW_CLI=1 monaco deploy -e environments.yaml 
```