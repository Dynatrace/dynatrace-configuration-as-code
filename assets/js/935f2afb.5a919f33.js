"use strict";(self.webpackChunkmonaco=self.webpackChunkmonaco||[]).push([[53],{1109:i=>{i.exports=JSON.parse('{"pluginId":"default","version":"current","label":"Next","banner":"unreleased","badge":true,"className":"docs-version-current","isLast":false,"docsSidebars":{"tutorialSidebar":[{"type":"link","label":"Monaco Documentation","href":"/dynatrace-configuration-as-code/next/","docId":"intro"},{"type":"category","label":"Get started","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"What is Monaco?","href":"/dynatrace-configuration-as-code/next/Get-started/intro","docId":"Get-started/intro"},{"type":"link","label":"Install Monaco","href":"/dynatrace-configuration-as-code/next/Get-started/installation","docId":"Get-started/installation"}]},{"type":"category","label":"Commands","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"Validating configuration","href":"/dynatrace-configuration-as-code/next/commands/validating-configuration","docId":"commands/validating-configuration"},{"type":"link","label":"Deploy projects","href":"/dynatrace-configuration-as-code/next/commands/deploying-projects","docId":"commands/deploying-projects"},{"type":"link","label":"Experimental New CLI","href":"/dynatrace-configuration-as-code/next/commands/experimental-new-cli","docId":"commands/experimental-new-cli"},{"type":"link","label":"Download configuration","href":"/dynatrace-configuration-as-code/next/commands/downloading-configuration","docId":"commands/downloading-configuration"},{"type":"link","label":"Logging","href":"/dynatrace-configuration-as-code/next/commands/logging","docId":"commands/logging"}]},{"type":"category","label":"Configuration","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"Deploy configuration","href":"/dynatrace-configuration-as-code/next/configuration/deploy_configuration","docId":"configuration/deploy_configuration"},{"type":"link","label":"Environments file","href":"/dynatrace-configuration-as-code/next/configuration/environments_file","docId":"configuration/environments_file"},{"type":"link","label":"Configuration structure","href":"/dynatrace-configuration-as-code/next/configuration/configuration_structure","docId":"configuration/configuration_structure"},{"type":"link","label":"Configuration YAML structure","href":"/dynatrace-configuration-as-code/next/configuration/yaml_config","docId":"configuration/yaml_config"},{"type":"link","label":"Plugin configuration","href":"/dynatrace-configuration-as-code/next/configuration/plugin_config","docId":"configuration/plugin_config"},{"type":"link","label":"Delete configuration","href":"/dynatrace-configuration-as-code/next/configuration/delete_config","docId":"configuration/delete_config"},{"type":"link","label":"Configuration types and token permissions","href":"/dynatrace-configuration-as-code/next/configuration/configTypes_tokenPermissions","docId":"configuration/configTypes_tokenPermissions"},{"type":"link","label":"Special Types of Configuration","href":"/dynatrace-configuration-as-code/next/configuration/special_config_types","docId":"configuration/special_config_types"}]},{"type":"category","label":"Guides","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"Add a new API","href":"/dynatrace-configuration-as-code/next/Guides/add_new_api","docId":"Guides/add_new_api"},{"type":"link","label":"Migrating deprecated configuration types","href":"/dynatrace-configuration-as-code/next/Guides/deprecated_migration","docId":"Guides/deprecated_migration"},{"type":"link","label":"Migration of application detection rules","href":"/dynatrace-configuration-as-code/next/Guides/app_detection_rules_migration","docId":"Guides/app_detection_rules_migration"},{"type":"link","label":"How to ensure ordering of application detection rules via monaco","href":"/dynatrace-configuration-as-code/next/Guides/ordering_app_detection_rules","docId":"Guides/ordering_app_detection_rules"}]},{"type":"category","label":"Useful links","collapsible":true,"collapsed":true,"items":[{"type":"link","label":"License and Bill of Materials","href":"/dynatrace-configuration-as-code/next/Useful-links/bill-of-materials","docId":"Useful-links/bill-of-materials"}]}]},"docs":{"commands/deploying-projects":{"id":"commands/deploying-projects","title":"Deploy projects","description":"The Monaco tool can deploy a configuration or a set of configurations in the form of projects.","sidebar":"tutorialSidebar"},"commands/downloading-configuration":{"id":"commands/downloading-configuration","title":"Download configuration","description":"This feature lets you download the configuration from a Dynatrace tenant as Monaco files.","sidebar":"tutorialSidebar"},"commands/experimental-new-cli":{"id":"commands/experimental-new-cli","title":"Experimental New CLI","description":"Monaco version 1.2.0+ includes the Beta version of the new CLI that is planned for a future release.","sidebar":"tutorialSidebar"},"commands/logging":{"id":"commands/logging","title":"Logging","description":"Use the MONACOREQUESTLOG and MONACORESPONSELOG environment variables to specify a file","sidebar":"tutorialSidebar"},"commands/validating-configuration":{"id":"commands/validating-configuration","title":"Validating configuration","description":"Monaco validates configuration files in a directory by performing a dry run.","sidebar":"tutorialSidebar"},"configuration/configTypes_tokenPermissions":{"id":"configuration/configTypes_tokenPermissions","title":"Configuration types and token permissions","description":"These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint.","sidebar":"tutorialSidebar"},"configuration/configuration_structure":{"id":"configuration/configuration_structure","title":"Configuration structure","description":"Configuration files are ordered by project in the projects folder.","sidebar":"tutorialSidebar"},"configuration/delete_config":{"id":"configuration/delete_config","title":"Delete configuration","description":"This guide shows you how to use the delete configuration tool to delete configuration that is not needed.","sidebar":"tutorialSidebar"},"configuration/deploy_configuration":{"id":"configuration/deploy_configuration","title":"Deploy configuration","description":"This guide will show you how to deploy a Monaco configuration to Dynatrace.","sidebar":"tutorialSidebar"},"configuration/environments_file":{"id":"configuration/environments_file","title":"Environments file","description":"The environments file is a YAML file used to define to which environment(s) to deploy configurations.","sidebar":"tutorialSidebar"},"configuration/plugin_config":{"id":"configuration/plugin_config","title":"Plugin configuration","description":"Important","sidebar":"tutorialSidebar"},"configuration/special_config_types":{"id":"configuration/special_config_types","title":"Special Types of Configuration","description":"Most types of configurations behave the same and entirely as described in other sections of this documentation.","sidebar":"tutorialSidebar"},"configuration/yaml_config":{"id":"configuration/yaml_config","title":"Configuration YAML structure","description":"This guide explains the structure of a YAML config file.","sidebar":"tutorialSidebar"},"Get-started/installation":{"id":"Get-started/installation","title":"Install Monaco","description":"This guide shows you how to download Monaco and install it on your operating system (Linux/macOS or Windows).","sidebar":"tutorialSidebar"},"Get-started/intro":{"id":"Get-started/intro","title":"What is Monaco?","description":"Monaco is a CLI tool that automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.","sidebar":"tutorialSidebar"},"Guides/add_new_api":{"id":"Guides/add_new_api","title":"Add a new API","description":"This guide shows you how to add a new API to Monaco that is not included in the table of supported APIs and how to determine whether an API is easy to add.","sidebar":"tutorialSidebar"},"Guides/app_detection_rules_migration":{"id":"Guides/app_detection_rules_migration","title":"Migration of application detection rules","description":"An internal change in how Dynatrace stores application detection rule configurations from version 1.252 upwards","sidebar":"tutorialSidebar"},"Guides/deprecated_migration":{"id":"Guides/deprecated_migration","title":"Migrating deprecated configuration types","description":"This guide shows you how to migrate deprecated configuration types.","sidebar":"tutorialSidebar"},"Guides/ordering_app_detection_rules":{"id":"Guides/ordering_app_detection_rules","title":"How to ensure ordering of application detection rules via monaco","description":"While the UI allows you to define the ordering of Application Detection Rules, it is not possible to define an order via monaco or the API.","sidebar":"tutorialSidebar"},"intro":{"id":"intro","title":"Monaco Documentation","description":"Monaco is a \\"monitoring as code\\" tool that allows you to create, update and version your monitoring configurations in Dynatrace efficiently and at scale.","sidebar":"tutorialSidebar"},"Useful-links/bill-of-materials":{"id":"Useful-links/bill-of-materials","title":"License and Bill of Materials","description":"Monaco is an open-source project released under the Apache 2.0 license.","sidebar":"tutorialSidebar"}}}')}}]);