---
sidebar_position: 2
title: Manage Configurations
---

With Monaco installed, you're ready to build your first monitoring configuration and have this deployed to your Dynatrace environment. To do so, you'll create some configuration files to define what is to be applied to your environment.
<p> </p>

## Prerequisites

<ul>
  <li>Monaco (2.0.0+) installed.</li>
  <li>A Dynatrace environment and access to create environment tokens.</li>
  <li>A Dynatrace token with <code>Write entities (entities.write)</code> permission. </li>
</ul>

<p> </p>

## Write Configuration

In this tutorial, you'll create a tagging rule in an environment that will apply tags to hosts and services where a Dynatrace server is detected.

Create a directory for your configuration:

```shell
$ mkdir learn-monaco-auto-tag
```

Change into the directory:
```shell
$ cd learn-monaco-auto-tag
```

Create a project directory to store the tagging configuration and change into the directory:
```shell
$ mkdir -p project-example/auto-tag; cd project-example/auto-tag
```

Create 2 files, the first file is to store the JSON configuration of the tagging configuration. The second file will be the [YAML configuration](../../configuration/yaml_configuration.md) file which lists the configurations to be deployed:
```shell
touch auto-tag.json auto-tag.yaml
```
<p></p>

Open `auto-tag.json` in your text editor, paste in the configuration below, then save the file


>**Tip** The name used in this configuration is specified as a variable and its value will be given in the YAML configuration

```json
{
    "name": "{{ .name }}",
    "rules": [
          {
                "type": "PROCESS_GROUP",
                "enabled": true,
                "valueFormat": null,
                "propagationTypes": [
                    "PROCESS_GROUP_TO_HOST",
                    "PROCESS_GROUP_TO_SERVICE"
                ],
                "conditions": [
                      {
                            "key": {
                                  "attribute": "PROCESS_GROUP_PREDEFINED_METADATA",
                                  "dynamicKey": "Dynatrace_CLUSTER_ID",
                                  "type": "PROCESS_PREDEFINED_METADATA_KEY"
                            },
                            "comparisonInfo": {
                                  "type": "STRING",
                                  "operator": "BEGINS_WITH",
                                  "value": "Server on Cluster",
                                  "negate": false,
                                  "caseSensitive": true
                            }
                      }
                ]
          }
    ]
}
```

Next, Open `auto-tag.yaml` in your text editor, paste in the configuration below, then save the file

>**Tip** The value of name is now provided:

```yaml
configs:
  - id: application-tagging
    config:
      template: "auto-tag.json"
      name: "DTServer"
```

<p></p>

Change back into the `learn-monaco-auto-tag` folder:
```shell
cd ../..
```

Create a [deployment manifest file](../../configuration/configuration.md#deployment-manifest) to instruct Monaco what project to deploy and exactly where it should be deployed:
```shell
touch deploy.yaml
```

Open the `deploy.yaml` file in your text editor and past in the configuration below, and save the file:

<p></p>

> **Replace*** the **url value** with your Dynatrace environment url

```yaml
projects:
  - name: auto-tag
    path: project-example

environments:
  - group: development
    entries:
      - name: development-environment
        url: "https://xxxxxx.live.Dynatrace.com"
        token:
          name: "development-token"
```
<p></p>

## Export Token to your Environment Variables

Export your Dynatrace token to your environment.

> **Tip** Ensure your token has **Write Permissions**

```shell
$ export development-token=YourTokenValue
```

## Validate Configuration

Ensure your configuration is syntactically valid and consistent by using the `monaco deploy --dry-run` command.

Validate your configuration. The example configuration provided above is valid, so Monaco will return a message finished without errors.

```shell
$ monaco  deploy --dry-run deploy.yaml
2021/10/13 13:22:06 INFO  Dynatrace Monitoring as Code v2.0.0
2021/10/13 13:22:06 INFO  Processing environment `development-environment`...
2021/10/13 13:22:06 INFO  Deployment finished without errors

```
