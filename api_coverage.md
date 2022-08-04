# Configuration API
| Endpoint                                                                 | Supported | Deprecated by Settings v2 |
|--------------------------------------------------------------------------|-----------|---------------------------|
| /api/config/v1/alertingProfiles                                          | ✔         | YES                       |
| /api/config/v1/anomalyDetection/applications                             | ✔️         |                           |
| /api/config/v1/anomalyDetection/aws                                      | ✔️         |                           |
| /api/config/v1/anomalyDetection/databaseServices                         | ✔️         |                           |
| /api/config/v1/anomalyDetection/diskEvents                               | ✔️         |                           |
| /api/config/v1/anomalyDetection/hosts                                    | NO        |                           |
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

# (TODO) Environment API v1 
| Endpoint                                                                 | Supported |
|--------------------------------------------------------------------------|-----------|
| /api/v1/synthetic/locations                                              | ✔️        |
| /api/v1/synthetic/monitors                                               | ✔️        |

# (TODO) Environment API v2
| Endpoint                                                                 | Supported |
|--------------------------------------------------------------------------|-----------|
| /api/v2/slo                                                              | ✔️        |
