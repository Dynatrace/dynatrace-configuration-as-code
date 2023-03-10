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

var StandardApiPropertyNameOfGetAllResponse = "values"

// configEndpointsV1 contains API definitions present in v1 to allow conversion and fallback deployment of v1
// This includes deprecated APIs removed with v2, as well as the '-v2' non-unique-name APIs moved to being the default
// and dropping the '-v2' suffix with v2.
var configEndpointsV1 = []API{
	{
		ID:                           "alerting-profile",
		URLPath:                      "/api/config/v1/alertingProfiles",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "management-zone",
		URLPath:                      "/api/config/v1/managementZones",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "auto-tag",
		URLPath:                      "/api/config/v1/autoTags",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "dashboard",
		URLPath:                      "/api/config/v1/dashboards",
		PropertyNameOfGetAllResponse: "dashboards",
		DeprecatedBy:                 "dashboard-v2",
	},
	{
		ID:                           "dashboard-v2",
		URLPath:                      "/api/config/v1/dashboards",
		PropertyNameOfGetAllResponse: "dashboards",
		NonUniqueName:                true,
	},
	{
		ID:                           "notification",
		URLPath:                      "/api/config/v1/notifications",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "extension",
		URLPath:                      "/api/config/v1/extensions",
		PropertyNameOfGetAllResponse: "extensions",
	},
	{
		ID:                  "extension-elasticsearch",
		URLPath:             "/api/config/v1/extensions/dynatrace.python.elasticsearch/global",
		SingleConfiguration: true,
	},
	{
		ID:                           "custom-service-java",
		URLPath:                      "/api/config/v1/service/customServices/java",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "custom-service-dotnet",
		URLPath:                      "/api/config/v1/service/customServices/dotNet",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "custom-service-go",
		URLPath:                      "/api/config/v1/service/customServices/go",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "custom-service-nodejs",
		URLPath:                      "/api/config/v1/service/customServices/nodeJS",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "custom-service-php",
		URLPath:                      "/api/config/v1/service/customServices/php",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "anomaly-detection-metrics",
		URLPath:                      "/api/config/v1/anomalyDetection/metricEvents",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "anomaly-detection-disks",
		URLPath:                      "/api/config/v1/anomalyDetection/diskEvents",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "synthetic-location",
		URLPath:                      "/api/v1/synthetic/locations",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "synthetic-monitor",
		URLPath:                      "/api/v1/synthetic/monitors",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "application",
		URLPath:                      "/api/config/v1/applications/web",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "application-web",
	},
	{
		ID:                           "application-web",
		URLPath:                      "/api/config/v1/applications/web",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "application-mobile",
		URLPath:                      "/api/config/v1/applications/mobile",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "app-detection-rule",
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
		ID:                           "aws-credentials",
		URLPath:                      "/api/config/v1/aws/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "kubernetes-credentials",
		URLPath:                      "/api/config/v1/kubernetes/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "azure-credentials",
		URLPath:                      "/api/config/v1/azure/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "request-attributes",
		URLPath:                      "/api/config/v1/service/requestAttributes",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "calculated-metrics-service",
		URLPath:                      "/api/config/v1/calculatedMetrics/service",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "calculated-metrics-log",
		URLPath:                      "/api/config/v1/calculatedMetrics/log",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "calculated-metrics-application-mobile",
		URLPath:                      "/api/config/v1/calculatedMetrics/mobile",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "calculated-metrics-synthetic",
		URLPath:                      "/api/config/v1/calculatedMetrics/synthetic",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "calculated-metrics-application-web",
		URLPath:                      "/api/config/v1/calculatedMetrics/rum",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "conditional-naming-processgroup",
		URLPath:                      "/api/config/v1/conditionalNaming/processGroup",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "conditional-naming-host",
		URLPath:                      "/api/config/v1/conditionalNaming/host",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "conditional-naming-service",
		URLPath:                      "/api/config/v1/conditionalNaming/service",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "maintenance-window",
		URLPath:                      "/api/config/v1/maintenanceWindows",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "request-naming-service",
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
		ID:                           "slo",
		URLPath:                      "/api/v2/slo",
		PropertyNameOfGetAllResponse: "slo",
	},
	{
		ID:                           "credential-vault",
		URLPath:                      "/api/config/v1/credentials",
		PropertyNameOfGetAllResponse: "credentials",
	},
	{
		ID:                           "failure-detection-parametersets",
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "failure-detection-rules",
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/rules",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "service-detection-full-web-request",
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "service-detection-full-web-service",
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "service-detection-opaque-web-request",
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "service-detection-opaque-web-service",
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                           "reports",
		URLPath:                      "/api/config/v1/reports",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                  "frequent-issue-detection",
		URLPath:             "/api/config/v1/frequentIssueDetection",
		SingleConfiguration: true,
	},
	{
		ID:                  "data-privacy",
		URLPath:             "/api/config/v1/dataPrivacy",
		SingleConfiguration: true,
	},
	{
		ID:                  "hosts-auto-update",
		URLPath:             "/api/config/v1/hosts/autoupdate",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-applications",
		URLPath:             "/api/config/v1/anomalyDetection/applications",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-aws",
		URLPath:             "/api/config/v1/anomalyDetection/aws",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-database-services",
		URLPath:             "/api/config/v1/anomalyDetection/databaseServices",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-hosts",
		URLPath:             "/api/config/v1/anomalyDetection/hosts",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-services",
		URLPath:             "/api/config/v1/anomalyDetection/services",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-vmware",
		URLPath:             "/api/config/v1/anomalyDetection/vmware",
		SingleConfiguration: true,
	},
	{
		ID:                  "service-resource-naming",
		URLPath:             "/api/config/v1/service/resourceNaming",
		SingleConfiguration: true,
	},
	{
		ID:                  "app-detection-rule-host",
		URLPath:             "/api/config/v1/applicationDetectionRules/hostDetection",
		SingleConfiguration: true,
	},
	{
		ID:                  "content-resources",
		URLPath:             "/api/config/v1/contentResources",
		SingleConfiguration: true,
	},
	{
		ID:                  "allowed-beacon-origins",
		URLPath:             "/api/config/v1/allowedBeaconOriginsForCors",
		SingleConfiguration: true,
	},
	{
		ID:                  "geo-ip-detection-headers",
		URLPath:             "/api/config/v1/geographicRegions/ipDetectionHeaders",
		SingleConfiguration: true,
	},
	{
		ID:                  "geo-ip-address-mappings",
		URLPath:             "/api/config/v1/geographicRegions/ipAddressMappings",
		SingleConfiguration: true,
	},
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
