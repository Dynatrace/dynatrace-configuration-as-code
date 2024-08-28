/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import "strings"

// configEndpointsV1 contains API definitions present in v1 to allow conversion and fallback deployment of v1
// This includes deprecated APIs removed with v2, as well as the '-v2' non-unique-name APIs moved to being the default
// and dropping the '-v2' suffix with v2.
var configEndpointsV1 = []API{
	{
		ID:                           AlertingProfile,
		URLPath:                      "/api/config/v1/alertingProfiles",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ManagementZone,
		URLPath:                      "/api/config/v1/managementZones",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           Autotag,
		URLPath:                      "/api/config/v1/autoTags",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           Dashboard,
		URLPath:                      "/api/config/v1/dashboards",
		PropertyNameOfGetAllResponse: "dashboards",
		DeprecatedBy:                 DashboardV2,
	},
	{
		ID:                           DashboardV2,
		URLPath:                      "/api/config/v1/dashboards",
		PropertyNameOfGetAllResponse: "dashboards",
		NonUniqueName:                true,
	},
	{
		ID:                           Notification,
		URLPath:                      "/api/config/v1/notifications",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           Extension,
		URLPath:                      "/api/config/v1/extensions",
		PropertyNameOfGetAllResponse: "extensions",
	},
	{
		ID:                  ExtensionElasticSearch,
		URLPath:             "/api/config/v1/extensions/dynatrace.python.elasticsearch/global",
		SingleConfiguration: true,
	},
	{
		ID:                           CustomServiceJava,
		URLPath:                      "/api/config/v1/service/customServices/java",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CustomServiceDotNet,
		URLPath:                      "/api/config/v1/service/customServices/dotNet",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CustomServiceGo,
		URLPath:                      "/api/config/v1/service/customServices/go",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CustomServiceNodeJs,
		URLPath:                      "/api/config/v1/service/customServices/nodeJS",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CustomServicePhp,
		URLPath:                      "/api/config/v1/service/customServices/php",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           AnomalyDetectionMetrics,
		URLPath:                      "/api/config/v1/anomalyDetection/metricEvents",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           AnomalyDetectionDisks,
		URLPath:                      "/api/config/v1/anomalyDetection/diskEvents",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           SyntheticLocation,
		URLPath:                      "/api/v1/synthetic/locations",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           SyntheticMonitor,
		URLPath:                      "/api/v1/synthetic/monitors",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "application",
		URLPath:                      "/api/config/v1/applications/web",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 ApplicationWeb,
	},
	{
		ID:                           ApplicationWeb,
		URLPath:                      "/api/config/v1/applications/web",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ApplicationMobile,
		URLPath:                      "/api/config/v1/applications/mobile",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           AppDetectionRule,
		URLPath:                      "/api/config/v1/applicationDetectionRules",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "app-detection-rule-v2",
	},
	{
		ID:                           "app-detection-rule-v2",
		URLPath:                      "/api/config/v1/applicationDetectionRules",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		NonUniqueName:                true,
	},
	{
		ID:                           AwsCredentials,
		URLPath:                      "/api/config/v1/aws/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           KubernetesCredentials,
		URLPath:                      "/api/config/v1/kubernetes/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           AzureCredentials,
		URLPath:                      "/api/config/v1/azure/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           RequestAttributes,
		URLPath:                      "/api/config/v1/service/requestAttributes",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CalculatedMetricsService,
		URLPath:                      "/api/config/v1/calculatedMetrics/service",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CalculatedMetricsLog,
		URLPath:                      "/api/config/v1/calculatedMetrics/log",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:logmonitoring.schemaless-log-metric",
	},
	{
		ID:                           CalculatedMetricsApplicationMobile,
		URLPath:                      "/api/config/v1/calculatedMetrics/mobile",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CalculatedMetricsSynthetic,
		URLPath:                      "/api/config/v1/calculatedMetrics/synthetic",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           CalculatedMetricsApplicationWeb,
		URLPath:                      "/api/config/v1/calculatedMetrics/rum",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ConditionalNamingProcessgroup,
		URLPath:                      "/api/config/v1/conditionalNaming/processGroup",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ConditionalNamingHost,
		URLPath:                      "/api/config/v1/conditionalNaming/host",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ConditionalNamingService,
		URLPath:                      "/api/config/v1/conditionalNaming/service",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           MaintenanceWindow,
		URLPath:                      "/api/config/v1/maintenanceWindows",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           RequestNamingService,
		URLPath:                      "/api/config/v1/service/requestNaming",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "request-naming-service-v2",
	},
	{
		ID:                           "request-naming-service-v2",
		URLPath:                      "/api/config/v1/service/requestNaming",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		NonUniqueName:                true,
	},
	{
		ID:                           Slo,
		URLPath:                      "/api/v2/slo",
		PropertyNameOfGetAllResponse: "slo",
	},
	{
		ID:                           CredentialVault,
		URLPath:                      "/api/config/v1/credentials",
		PropertyNameOfGetAllResponse: "credentials",
	},
	{
		ID:                           FailureDetectionParametersets,
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           FailureDetectionRules,
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/rules",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ServiceDetectionFullWebRequest,
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ServiceDetectionFullWebService,
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ServiceDetectionOpaqueWebRequest,
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           ServiceDetectionOpaqueWebService,
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           Reports,
		URLPath:                      "/api/config/v1/reports",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                  FrequentIssueDetection,
		URLPath:             "/api/config/v1/frequentIssueDetection",
		SingleConfiguration: true,
	},
	{
		ID:                  DataPrivacy,
		URLPath:             "/api/config/v1/dataPrivacy",
		SingleConfiguration: true,
	},
	{
		ID:                  HostsAutoUpdate,
		URLPath:             "/api/config/v1/hosts/autoupdate",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionApplications,
		URLPath:             "/api/config/v1/anomalyDetection/applications",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionAws,
		URLPath:             "/api/config/v1/anomalyDetection/aws",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionDatabaseServices,
		URLPath:             "/api/config/v1/anomalyDetection/databaseServices",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionHosts,
		URLPath:             "/api/config/v1/anomalyDetection/hosts",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionServices,
		URLPath:             "/api/config/v1/anomalyDetection/services",
		SingleConfiguration: true,
	},
	{
		ID:                  AnomalyDetectionVmware,
		URLPath:             "/api/config/v1/anomalyDetection/vmware",
		SingleConfiguration: true,
	},
	{
		ID:                  ServiceResourceNaming,
		URLPath:             "/api/config/v1/service/resourceNaming",
		SingleConfiguration: true,
	},
	{
		ID:                  AppDetectionRuleHost,
		URLPath:             "/api/config/v1/applicationDetectionRules/hostDetection",
		SingleConfiguration: true,
	},
	{
		ID:                  ContentResources,
		URLPath:             "/api/config/v1/contentResources",
		SingleConfiguration: true,
	},
	{
		ID:                  AllowedBeaconOrigins,
		URLPath:             "/api/config/v1/allowedBeaconOriginsForCors",
		SingleConfiguration: true,
	},
	{
		ID:                  GeoIpDetectionHeaders,
		URLPath:             "/api/config/v1/geographicRegions/ipDetectionHeaders",
		SingleConfiguration: true,
	},
	{
		ID:                  GeoIpAddressMappings,
		URLPath:             "/api/config/v1/geographicRegions/ipAddressMappings",
		SingleConfiguration: true,
	},
}

// NewV1APIs returns collection of predefined API to work with Dynatrace
// Deprecated: Please use NewAPIs. This one is legacy and is used only to convert old to new types of APIs
func NewV1APIs() APIs {
	return newAPIs(configEndpointsV1)
}

// GetV2ID returns the ID of APIs in v2 - replacing deprecated APIs with their new version and dropping the -v2 marker
// from APIs introducing the breaking change of handling non-unique-names. This is used in v1 -> v2 conversion
func GetV2ID(forV1Api API) string {
	n := forV1Api.ID
	if forV1Api.DeprecatedBy != "" {
		n = forV1Api.DeprecatedBy
	}
	return strings.TrimSuffix(n, "-v2")
}
