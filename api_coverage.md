# Configuration API

| Endpoint                                                                 | Supported | Deprecated by Settings v2 |
|--------------------------------------------------------------------------|-----------|---------------------------|
| /api/config/v1/alertingProfiles                                          | ✔         | YES                       |
| /api/config/v1/anomalyDetection/applications                             | ✔️         |                           |
| /api/config/v1/anomalyDetection/aws                                      | ✔️         |                           |
| /api/config/v1/anomalyDetection/databaseServices                         | ✔️         |                           |
| /api/config/v1/anomalyDetection/diskEvents                               | ✔️         |                           |
| /api/config/v1/anomalyDetection/hosts                                    | ✔️         |                           |
| /api/config/v1/anomalyDetection/metricEvents                             | ✔         |                           | 
| /api/config/v1/anomalyDetection/processGroups                            | NO        |                           |
| /api/config/v1/anomalyDetection/services                                 | ✔️         |                           |
| /api/config/v1/anomalyDetection/vmware                                   | ✔️         |                           |
| /api/config/v1/autoTags                                                  | ✔️         |                           |
| /api/config/v1/aws/credentials                                           | ✔️         |                           |
| /api/config/v1/aws/privatelink                                           | NO️        |                           |
| /api/config/v1/azure/credentials                                         | ✔️         |                           |
| /api/config/v1/calculatedMetrics/mobile                                  | ✔️         |                           |
| /api/config/v1/calculatedMetrics/rum                                     | ✔️         |                           |
| /api/config/v1/calculatedMetrics/log                                     | ✔️         |                           |
| /api/config/v1/calculatedMetrics/service                                 | ✔️         |                           |
| /api/config/v1/calculatedMetrics/synthetic                               | ✔️         |                           |
| /api/config/v1/cloudFoundry/credentials                                  | NO️        |                           |
| /api/config/v1/conditionalNaming/host                                    | ✔️         |                           |
| /api/config/v1/conditionalNaming/processGroup                            | ✔️         |                           |
| /api/config/v1/conditionalNaming/service                                 | ✔️         |                           |
| /api/config/v1/credentials                                               | ✔️         |                           |
| /api/config/v1/dashboards                                                | ✔️         |                           |
| /api/config/v1/dataPrivacy                                               | ✔️         |                           |
| /api/config/v1/extensions                                                | ✔️         |                           |
| /api/config/v1/extensions/dynatrace.python.elasticsearch/global          | ✔️         |                           |
| /api/config/v1/frequentIssueDetection                                    | ✔️         | YES                       |
| /api/config/v1/kubernetes/credentials                                    | ✔️         |                           |
| /api/config/v1/maintenanceWindows                                        | ✔️         | YES                       |
| /api/config/v1/managementZones                                           | ✔️         |                           |
| /api/config/v1/symfiles                                                  | NO        |                           |
| /api/config/v1/notifications                                             | ✔️         | YES                       |
| /api/config/v1/hosts/autoupdate                                          | ✔️         |                           |
| /api/config/v1/hostgroups/{id}                                           | NO️        |                           |
| /api/config/v1/hosts/{id}                                                | NO️        |                           |
| /api/config/v1/plugins                                                   | NO️        |                           |
| /api/config/v1/remoteEnvironments                                        | NO️        |                           |
| /api/config/v1/reports                                                   | ✔️         |                           |
| /api/config/v1/allowedBeaconOriginsForCors                               | ✔️         |                           |
| /api/config/v1/applicationDetectionRules                                 | ✔️         |                           |
| /api/config/v1/applicationDetectionRules/hostDetection                   | ✔️         |                           |
| /api/config/v1/contentResources                                          | ✔️         |                           |
| /api/config/v1/geographicRegions/ipDetectionHeaders                      | ✔️         |                           |
| /api/config/v1/geographicRegions/ipAddressMappings                       | ✔️         |                           |
| /api/config/v1/applications/mobile                                       | ✔️         |                           |
| /api/config/v1/applications/web                                          | ✔️         |                           |
| /api/config/v1/service/customServices/java                               | ✔️         |                           |
| /api/config/v1/service/customServices/dotnet                             | ✔️         |                           |
| /api/config/v1/service/customServices/go                                 | ✔️         |                           |
| /api/config/v1/service/customServices/nodejs                             | ✔️         |                           |
| /api/config/v1/service/customServices/php                                | ✔️         |                           |
| /api/config/v1/service/detectionRules/FULL_WEB_REQUEST                   | ✔️         |                           |
| /api/config/v1/service/detectionRules/FULL_WEB_SERVICE                   | ✔️         |                           |
| /api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST    | ✔️         |                           |
| /api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE    | ✔️         |                           |
| /api/config/v1/service/failureDetection/parameterSelection/parameterSets | ✔️         |                           |
| /api/config/v1/service/failureDetection/parameterSelection/rules         | ✔️         |                           |
| /api/config/v1/service/failureDetection/ibmMqTracing                     | NO️        | YES                       |
| /api/config/v1/service/requestAttributes                                 | ✔️         |                           |
| /api/config/v1/service/requestNaming                                     | ✔️         |                           |
| /api/config/v1/service/resourceNaming                                    | ✔️         |                           |

# Environment API v1

Most environment APIs are not 'configuration' - this list only contains APIs that clearly fall into scope of monitoring
config as code.

| Endpoint                                                                 | Supported | Deprecated by Settings v2 |
|--------------------------------------------------------------------------|-----------|---------------------------|
| /api/v1/synthetic/locations                                              | ✔         |                           |
| /api/v1/synthetic/monitors                                               | ✔         |                           |

Topology & Smartscape APIs *might* fall into scope of monaco - however they can be used to manually set tags on
monitored
entities, and will need dedicated implementation to work.

# Environment API v2

| Endpoint                            | Description                                     | Supported | Deprecated by Settings v2 |
|-------------------------------------|-------------------------------------------------|-----------|---------------------------|
| /api/v2/slo                         | SLOs                                            | ✔         |                           |
| /api/v2/activeGates/{id}/autoUpdate | Auto-update config for specific env active gate | NO        |                           |
| /api/v2/activeGates/autoUpdate      | Global uto-update config for env active gates   | NO        |                           |
| /api/v2/extensions                  | Extension 2.0 upload and configuration          | NO        |                           |
| /api/v2/networkZones                | Specific and global networkzone settings        | NO        |                           |
| /api/v2/settings                    | Settings 2.0                                    | NO        |                           |
| /api/v2/synthetic                   | Global synthetic settings and v2 locations API  | NO        |                           |

# Summarized List of currently unsupported Config APIs

| Endpoint                                             | Deprecated by Settings v2 | Default API Pattern                    |
|------------------------------------------------------|---------------------------|----------------------------------------|
| /api/v2/activeGates/{id}/autoUpdate                  |                           | X                                      |
| /api/v2/activeGates/autoUpdate                       |                           | X                                      |
| /api/v2/extensions                                   |                           | X - but similar to v1 extensions       |
| /api/v2/networkZones                                 |                           | ~ (no POST)                            |
| /api/v2/settings                                     |                           | X                                      |
| /api/v2/synthetic                                    |                           | ✔                                      |
| /api/config/v1/service/failureDetection/ibmMqTracing | YES                       |                                        | 
| /api/config/v1/hostgroups/{id}                       |                           | X (hostgroup ID needs to be known)     |  
| /api/config/v1/hosts/{id}                            |                           | X (host ID needs to be known)          |  
| /api/config/v1/plugins                               |                           | X - but similar to v1 extensions       |
| /api/config/v1/remoteEnvironments                    |                           | ✔                                      |
| /api/config/v1/symfiles                              |                           | X                                      |
| /api/config/v1/cloudFoundry/credentials              |                           | ✔                                      |
| /api/config/v1/aws/privatelink                       |                           | X                                      |
| /api/config/v1/anomalyDetection/processGroups        |                           | X (process group ID needs to be known) | 


