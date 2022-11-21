---
sidebar_position: 1
title: Manage configurations
---

To get you started with managing configurations, this section will guide you through a simple example: creating a tagging rule with Monaco. 
You will learn how to create, deploy, and delete a configuration. 

** Prerequisites **

* Monaco 2.0.0+ installed (see [Install Monaco](../Get-Started/install-monaco.mdx))
* A Dynatrace environment and access to create environment tokens
* A Dynatrace token with the following permissions: 
    * Write entities (ApV1) <code>entities.write</code> 
    * Write configuration (ApV2) <code>WriteConfig</code> 

>
> :warning: To learn how to create tokens, please refer to the [Dynatrace documentation](https://www.dynatrace.com/support/help/shortlink/token#create-api-token). 
>

## Create configuration

In this tutorial, you'll create a tagging rule in an environment that will apply tags to hosts and services where a Dynatrace server is detected.

1\. Open your preferred CLI. 

2\. Create a directory for your configuration

```shell
mkdir learn-monaco-auto-tag
```

3\. Change to the new directory

```shell
cd learn-monaco-auto-tag
```

4\. Create a project directory to store the tagging configuration and change to it

```shell
mkdir -p project-example/auto-tag
```

```shell
cd project-example/auto-tag
```

5\. Create two files. 
* The first file (auto-tag.json) stores the JSON configuration of the tagging configuration. 
* The second file (auto-tag.yaml) will be the [YAML configuration](../../configuration/yaml_configuration.md) file which lists the configurations to be deployed

```shell
touch auto-tag.json auto-tag.yaml
```

6\. Open the JSON configuration file (`auto-tag.json`) in your text editor and paste in the configuration below. Save the file


>
> :warning: The name used in this configuration is specified as a variable and its value will be given in the YAML configuration
>

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
                                  "dynamicKey": "DYNATRACE_CLUSTER_ID",
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

7\. Open the YAML configuration file (`auto-tag.yaml`) in your text editor and paste the configuration below. Save the file

>
> :warning: The name of the tag should be provided here. In this example: `DTServer`
>

```yaml
configs:
  - id: application-tagging
    type: 
      api: auto-tag
    config:
      template: "auto-tag.json"
      name: "DTServer"
```

<p></p>

8\. Change back to the configuration directory folder (`learn-monaco-auto-tag`)

```shell
cd ../..
```

9\. Create a [deployment manifest file](../../configuration/configuration.md#deployment-manifest) to instruct Monaco what project to deploy and where to deploy it

```shell
touch deploy.yaml
```

10\. Open the deployment manifest file (`deploy.yaml`) in your text editor and paste in the configuration below. Save the file

> :warning: **Replace the URL value** with the URL of your Dynatrace environment

```yaml
manifest_version: 1.0

projects:
  - name: auto-tag
    path: project-example

environments:
  - group: development
    entries:
      - name: development-environment
        url: 
          value: "https://xxxxxx.live.Dynatrace.com"
        token:
          name: "devToken"
```

11\. Export your Dynatrace token to your environment

>
> :warning: Ensure your token has permissions to **write entities** and **write configuration**
>

```shell
export devToken=YourTokenValue
```

12\. Run `monaco deploy --dry-run` to ensure your configuration is syntactically valid and consistent 

```shell
monaco  deploy --dry-run deploy.yaml
```

13\. If the dry run was successful, Monaco will return the following message:

```shell
2021/10/13 13:22:06 INFO  Dynatrace Monitoring as Code v2.0.0
2021/10/13 13:22:06 INFO  Processing environment `development-environment`...
2021/10/13 13:22:06 INFO  Deployment finished without errors
```

## Deploy configuration

Now that you have [created the configuration](./manage-configuration#create-configuration), you need to deploy it to your Dynatrace environment. To do this, you use the `monaco deploy` command.

1\. To apply your configuration with the `monaco deploy` command, provide the name of the deployment file as argument 

```shell
monaco  deploy deploy.yaml
```

2\. If the deployment is successful, Monaco will return the following message:

```shell
2021/10/13 14:48:43 INFO  Dynatrace Monitoring as Code v2.0.0
2021/10/13 14:48:43 INFO  Processing environment `development-environment`...
2021/10/13 14:48:43 INFO  Deployment finished without errors
```
>
> :warning: If your configuration fails to deploy, you may have syntax errors in your files or your token requires more permissions. 
> Please refer to the output error description.
>

3\. To check if your tag has been created, open your Dynatrace environment in your browser

4\. Go to Manage > Settings > Tags > Automatically Applied Tags

5\. Search for `DTServer`

<!-- <img
  src={require('../static/img/DTServer.PNG').default}
  alt="dtserver"
/> -->

## Delete configuration

Now that your configuration is [deployed](./manage-configuration#deploy-configuration), you can delete it. To do this, you will use the `monaco delete` command.

1\. To delete the previously created tag `DTServer`, create a file called `delete.yaml` in the root folder

```shell
touch delete.yaml
```

2\. Open the file in your text editor and paste the following configuration, then save the changes.

```yaml
delete:
  - "auto-tag/DTserver"
```

3\. Delete the tag from your environment. 

>
> :warning: The `delete` command requires both the **delete file** and the **deployment manifest file** as arguments. The deployment manifest file indicates in which environment(s) to delete the configuration specified in the delete file. 
>

```shell
monaco delete deploy.yaml delete.yaml
```
4\. If the deletion was successful, Monaco will return the following message:

```shell
2021/10/13 18:21:33 INFO  Dynatrace Monitoring as Code v2.0.0
```
