"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[8219],{9033:i=>{i.exports=JSON.parse('{"pluginId":"default","version":"1.6.0","label":"1.6.0","banner":"unmaintained","badge":true,"className":"docs-version-1.6.0","isLast":false,"docsSidebars":{"version-1.6.0/tutorialSidebar":[{"type":"link","label":"What is Monaco?","href":"/dynatrace-monitoring-as-code/1.6.0/","docId":"intro"},{"type":"link","label":"Install Monaco","href":"/dynatrace-monitoring-as-code/1.6.0/installation","docId":"installation"},{"type":"category","label":"Commands","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"Validating Configuration","href":"/dynatrace-monitoring-as-code/1.6.0/commands/validating-configuration","docId":"commands/validating-configuration"},{"type":"link","label":"Deploying Projects","href":"/dynatrace-monitoring-as-code/1.6.0/commands/deploying-projects","docId":"commands/deploying-projects"},{"type":"link","label":"Experimental New CLI","href":"/dynatrace-monitoring-as-code/1.6.0/commands/experimental-new-cli","docId":"commands/experimental-new-cli"},{"type":"link","label":"Downloading Configuration","href":"/dynatrace-monitoring-as-code/1.6.0/commands/downloading-configuration","docId":"commands/downloading-configuration"},{"type":"link","label":"Logging","href":"/dynatrace-monitoring-as-code/1.6.0/commands/Logging","docId":"commands/Logging"}]},{"type":"category","label":"Configuration","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"Deploying Configuration to Dynatrace","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/deploy_configuration","docId":"configuration/deploy_configuration"},{"type":"link","label":"Environments file","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/environments_file","docId":"configuration/environments_file"},{"type":"link","label":"Configuration Structure","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/configuration_structure","docId":"configuration/configuration_structure"},{"type":"link","label":"Configuration YAML Structure","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/yaml_confg","docId":"configuration/yaml_confg"},{"type":"link","label":"Plugin Configuration","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/plugin_config","docId":"configuration/plugin_config"},{"type":"link","label":"Delete Configuration","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/delete_config","docId":"configuration/delete_config"},{"type":"link","label":"Configuration Types and Token Permissions","href":"/dynatrace-monitoring-as-code/1.6.0/configuration/configTypes_tokenPermissions","docId":"configuration/configTypes_tokenPermissions"}]},{"type":"category","label":"Guides","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"How to add a new API","href":"/dynatrace-monitoring-as-code/1.6.0/Guides/add_new_api","docId":"Guides/add_new_api"}]},{"type":"link","label":"License and Bill of material","href":"/dynatrace-monitoring-as-code/1.6.0/license","docId":"License and Bill of materials"}]},"docs":{"commands/deploying-projects":{"id":"commands/deploying-projects","title":"Deploying Projects","description":"The tool allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to a environment or a group of environments. This is done by passing the --project flag (or -p for short).","sidebar":"version-1.6.0/tutorialSidebar"},"commands/downloading-configuration":{"id":"commands/downloading-configuration","title":"Downloading Configuration","description":"This feature allows you to download the configuration from a Dynatrace tenant as Monaco files. You can use this feature to avoid starting from scratch when using Monaco. For this feature you will have to enable CLI version 2.0.","sidebar":"version-1.6.0/tutorialSidebar"},"commands/experimental-new-cli":{"id":"commands/experimental-new-cli","title":"Experimental New CLI","description":"Starting with version 1.2.0 a new experimental CLI is available. The plan is that it will gradually become the new default in the next few releases.","sidebar":"version-1.6.0/tutorialSidebar"},"commands/Logging":{"id":"commands/Logging","title":"Logging","description":"Sometimes it is useful for debugging to see http traffic between monaco and the dynatrace api. This is possible by specifying a log file via the MONACOREQUESTLOG and MONACORESPONSELOG env variables.","sidebar":"version-1.6.0/tutorialSidebar"},"commands/validating-configuration":{"id":"commands/validating-configuration","title":"Validating Configuration","description":"Monaco validates the configuration files in a directory, it does so by performing a dry run. It will check whether your Dynatrace config files are valid JSON, and whether your tool configuration yaml files can be parsed and used.","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/configTypes_tokenPermissions":{"id":"configuration/configTypes_tokenPermissions","title":"Configuration Types and Token Permissions","description":"These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint.","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/configuration_structure":{"id":"configuration/configuration_structure","title":"Configuration Structure","description":"Configuration files are ordered by project in the projects folder. Project folder can only contain:","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/delete_config":{"id":"configuration/delete_config","title":"Delete Configuration","description":"Configuration which is not needed anymore can also be deleted in automated fashion. This tool is looking for delete.yaml file located in projects root","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/deploy_configuration":{"id":"configuration/deploy_configuration","title":"Deploying Configuration to Dynatrace","description":"Monaco allows for deploying a configuration or a set of configurations in the form of project(s). A project is a folder containing files that define configurations to be deployed to a environment or a group of environments. This is done by passing the --project flag (or -p for short).","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/environments_file":{"id":"configuration/environments_file","title":"Environments file","description":"Environments are defined in the environments.yaml consisting of the environment url and the name of the environment variable to use for the API token.","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/plugin_config":{"id":"configuration/plugin_config","title":"Plugin Configuration","description":"Important","sidebar":"version-1.6.0/tutorialSidebar"},"configuration/yaml_confg":{"id":"configuration/yaml_confg","title":"Configuration YAML Structure","description":"Every configuration needs a YAML containing required and optional content.","sidebar":"version-1.6.0/tutorialSidebar"},"Guides/add_new_api":{"id":"Guides/add_new_api","title":"How to add a new API","description":"You spotted a new API which you want to automate using monaco, but sadly it\'s not in the","sidebar":"version-1.6.0/tutorialSidebar"},"installation":{"id":"installation","title":"Install Monaco","description":"To use monaco you will need to install it. Monaco is distributed as a binary package.","sidebar":"version-1.6.0/tutorialSidebar"},"intro":{"id":"intro","title":"What is Monaco?","description":"Monaco is CLI tool that automates deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.","sidebar":"version-1.6.0/tutorialSidebar"},"License and Bill of materials":{"id":"License and Bill of materials","title":"License and Bill of material","description":"Monaco is an open-source project released under the Apache 2.0 license.","sidebar":"version-1.6.0/tutorialSidebar"}}}')}}]);