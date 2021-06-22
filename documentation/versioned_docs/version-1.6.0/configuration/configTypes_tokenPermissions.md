---
sidebar_position: 7
---

# Configuration Types and Token Permissions

These are the supported configuration types, their API endpoints and the token permissions required for interacting with any of endpoint.

| Configuration                   | Endpoint                                        | Token Permission(s)                                                                                                 |
| ------------------------------- | ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| alerting-profile                | _/api/config/v1/alertingProfiles_               | `Read Configuration` & `Write Configuration`                                                                        |
| anomaly-detection-metrics       | _/api/config/v1/anomalyDetection/metricEvents_  | `Read Configuration` & `Write Configuration`                                                                        |
| app-detection-rule              | _/api/config/v1/applicationDetectionRules_      | `Read Configuration` & `Write Configuration`                                                                        |
| application **deprecated in 2.0.0!**| _/api/config/v1/applications/web_           | `Read Configuration` & `Write Configuration`                                                                        |
| application-web **replaces application**| _/api/config/v1/applications/web_       | `Read Configuration` & `Write Configuration`                                                                        |
| application-mobile              | _/api/config/v1/applications/mobile_            | `Read Configuration` & `Write Configuration`                                                                        |
| auto-tag                        | _/api/config/v1/autoTags_                       | `Read Configuration` & `Write Configuration`                                                                        |
| aws-credentials                 | _/api/config/v1/aws/credentials_                | `Read Configuration` & `Write Configuration`                                                                        |
| azure-credentials               | _/api/config/v1/azure/credentials_              | `Read Configuration` & `Write Configuration`                                                                        |
| calculated-metrics-log          | _/api/config/v1/calculatedMetrics/log_          | `Read Configuration` & `Write Configuration`                                                                        |
| calculated-metrics-service      | _/api/config/v1/calculatedMetrics/service_      | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-host         | _/api/config/v1/conditionalNaming/host_         | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-processgroup | _/api/config/v1/conditionalNaming/processGroup_ | `Read Configuration` & `Write Configuration`                                                                        |
| conditional-naming-service      | _/api/config/v1/conditionalNaming/service_      | `Read Configuration` & `Write Configuration`                                                                        |
| credential-vault                | _/api/config/v1/credentials_                    | `Read Credential Vault Entries` & `Write Credential Vault Entries`                                                  |
| custom-service-java             | _/api/config/v1/service/customServices/java_    | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-dotnet           | _/api/config/v1/service/customServices/dotnet_  | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-go               | _/api/config/v1/service/customServices/go_      | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-nodejs           | _/api/config/v1/service/customServices/nodejs_  | `Read Configuration` & `Write Configuration`                                                                        |
| custom-service-php              | _/api/config/v1/service/customServices/php_     | `Read Configuration` & `Write Configuration`                                                                        |
| dashboard                       | _/api/config/v1/dashboards_                     | `Read Configuration` & `Write Configuration`                                                                        |
| extension                       | _/api/config/v1/extensions_                     | `Read Configuration` & `Write Configuration`                                                                        |
| failure-detection-parametersets          | _/api/config/v1/service/failureDetection/parameterSelection/parameterSets_  | `Read Configuration` & `Write Configuration`                                   |
| failure-detection-rules                  | _/api/config/v1/service/failureDetection/parameterSelection/rules_          | `Read Configuration` & `Write Configuration`                                   |
| kubernetes-credentials          | _/api/config/v1/kubernetes/credentials_         | `Read Configuration` & `Write Configuration`                                                                        |
| maintenance-window              | _/api/config/v1/maintenanceWindows_             | `Read Configuratio`  & `Write Configuration`                                                                                     |
| management-zone                 | _/api/config/v1/managementZones_                | `Read Configuration` & `Write Configuration`                                                                        |
| notification                    | _/api/config/v1/notifications_                  | `Read Configuration` & `Write Configuration`                                                                        |
| request-attributes              | _/api/config/v1/service/requestAttributes_      | `Read Configuration` & `Capture request data`                                                                       |
| request-naming-service          | _/api/config/v1/service/requestNaming_          | `Read Configuration` & `Write Configuration`                                                                        |
| slo                             | _/api/v2/slo_                                   | `Read SLO` & `Write SLOs`                                                                                           |
| synthetic-location              | _/api/v1/synthetic/locations_                   | `Access problem and event feed, metrics, and topology` & `Create and read synthetic monitors, locations, and nodes` |
| synthetic-monitor               | _/api/v1/synthetic/monitors_                    | `Create and read synthetic monitors, locations, and nodes`                                                          |

For reference, refer to [this](https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication) page for a detailed
description to each token permission.

If your desired API is not in the table above, please consider adding it be following the instructions in [How to add new APIs](Guides/add_new_api.md).
