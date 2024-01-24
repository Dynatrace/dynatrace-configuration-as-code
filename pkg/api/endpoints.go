/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

// configEndpoints is map of the http endpoints for configuration API (aka classic/config endpoints).
var configEndpoints = []API{
	{
		ID:                           "alerting-profile",
		URLPath:                      "/api/config/v1/alertingProfiles",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:alerting.profile",
		NonUniqueName:                true,
	},
	{
		ID:                           "network-zone",
		URLPath:                      "/api/v2/networkZones",
		PropertyNameOfGetAllResponse: "networkZones",
		TweakResponseFunc: func(m map[string]any) {
			delete(m, "numOfOneAgentsUsing")
			delete(m, "numOfConfiguredOneAgents")
			delete(m, "numOfOneAgentsFromOtherZones")
			delete(m, "numOfConfiguredActiveGates")
		},
	},
	{
		ID:                           "management-zone",
		URLPath:                      "/api/config/v1/managementZones",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:management-zones",
	},
	{
		ID:                           "auto-tag",
		URLPath:                      "/api/config/v1/autoTags",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:tags.auto-tagging",
	},
	{
		ID:                           "dashboard",
		URLPath:                      "/api/config/v1/dashboards",
		PropertyNameOfGetAllResponse: "dashboards",
		NonUniqueName:                true,
	},
	{
		ID:                           "notification",
		URLPath:                      "/api/config/v1/notifications",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:problem.notifications",
	},
	{
		ID:                           "extension",
		URLPath:                      "/api/config/v1/extensions",
		PropertyNameOfGetAllResponse: "extensions",
		SkipDownload:                 true,
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
		DeprecatedBy:                 "builtin:anomaly-detection.metric-events",
		NonUniqueName:                true,
	},
	// Early adopter API !
	{
		ID:                           "anomaly-detection-disks",
		URLPath:                      "/api/config/v1/anomalyDetection/diskEvents",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:anomaly-detection.infrastructure-disks",
	},
	// Environment API not Config API
	{
		ID:                           "synthetic-location",
		URLPath:                      "/api/v1/synthetic/locations",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	// Environment API not Config API
	{
		ID:                           "synthetic-monitor",
		URLPath:                      "/api/v1/synthetic/monitors",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
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
		DeprecatedBy:                 "builtin:rum.web.app-detection",
	},
	{
		ID:                           "aws-credentials",
		URLPath:                      "/api/config/v1/aws/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		SkipDownload:                 true,
	},
	// Early adopter API !
	{
		ID:                           "kubernetes-credentials",
		URLPath:                      "/api/config/v1/kubernetes/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:cloud.kubernetes",
		//NonUniqueName: true, // non-unique name handling for k8s credentials does not work, as path ID needs to be a ME-ID not a uuid; handling as unique again for now
		SkipDownload: true,
	},
	{
		ID:                           "azure-credentials",
		URLPath:                      "/api/config/v1/azure/credentials",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		SkipDownload:                 true,
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
	// Early adopter API !
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
		DeprecatedBy:                 "builtin:alerting.maintenance-window",
	},
	{
		ID:                           "request-naming-service",
		URLPath:                      "/api/config/v1/service/requestNaming",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		NonUniqueName:                true,
	},
	// Environment API not Config API
	{
		ID:                           "slo",
		URLPath:                      "/api/v2/slo",
		PropertyNameOfGetAllResponse: "slo",
	},
	{
		ID:                           "credential-vault",
		URLPath:                      "/api/config/v1/credentials",
		PropertyNameOfGetAllResponse: "credentials",
		SkipDownload:                 true,
	},
	{
		ID:                           "failure-detection-parametersets",
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:failure-detection.environment.parameters",
	},
	{
		ID:                           "failure-detection-rules",
		URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/rules",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:failure-detection.environment.rules",
	},
	{
		ID:                           "service-detection-full-web-request",
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:service-detection.full-web-request",
	},
	{
		ID:                           "service-detection-full-web-service",
		URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:service-detection.full-web-service",
	},
	{
		ID:                           "service-detection-opaque-web-request",
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:service-detection.external-web-request",
	},
	{
		ID:                           "service-detection-opaque-web-service",
		URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		DeprecatedBy:                 "builtin:service-detection.external-web-service",
	},
	// Early adopter API !
	{
		ID:                           "reports",
		URLPath:                      "/api/config/v1/reports",
		PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
	},
	{
		ID:                  "frequent-issue-detection",
		URLPath:             "/api/config/v1/frequentIssueDetection",
		DeprecatedBy:        "builtin:anomaly-detection.frequent-issues",
		SingleConfiguration: true,
	},
	{
		ID:                  "data-privacy",
		URLPath:             "/api/config/v1/dataPrivacy",
		DeprecatedBy:        "builtin:preferences.privacy",
		SingleConfiguration: true,
	},
	{
		ID:                  "hosts-auto-update",
		URLPath:             "/api/config/v1/hosts/autoupdate",
		DeprecatedBy:        "builtin:deployment.oneagent.updates",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-applications",
		URLPath:             "/api/config/v1/anomalyDetection/applications",
		DeprecatedBy:        "builtin:anomaly-detection.rum-web, builtin:anomaly-detection.rum-mobile",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-aws",
		URLPath:             "/api/config/v1/anomalyDetection/aws",
		DeprecatedBy:        "builtin:anomaly-detection.infrastructure-aws",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-database-services",
		URLPath:             "/api/config/v1/anomalyDetection/databaseServices",
		DeprecatedBy:        "builtin:anomaly-detection.databases",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-hosts",
		URLPath:             "/api/config/v1/anomalyDetection/hosts",
		DeprecatedBy:        "builtin:anomaly-detection.infrastructure-hosts",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-services",
		URLPath:             "/api/config/v1/anomalyDetection/services",
		DeprecatedBy:        "builtin:anomaly-detection.services",
		SingleConfiguration: true,
	},
	{
		ID:                  "anomaly-detection-vmware",
		URLPath:             "/api/config/v1/anomalyDetection/vmware",
		DeprecatedBy:        "builtin:anomaly-detection.infrastructure-vmware",
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
		DeprecatedBy:        "builtin:rum.host-headers",
		SingleConfiguration: true,
	},
	{
		ID:                  "content-resources",
		URLPath:             "/api/config/v1/contentResources",
		DeprecatedBy:        "builtin:rum.provider-breakdown",
		SingleConfiguration: true,
	},
	{
		ID:                  "allowed-beacon-origins",
		URLPath:             "/api/config/v1/allowedBeaconOriginsForCors",
		DeprecatedBy:        "builtin:rum.web.beacon-domain-origins",
		SingleConfiguration: true,
	},
	{
		ID:                  "geo-ip-detection-headers",
		URLPath:             "/api/config/v1/geographicRegions/ipDetectionHeaders",
		DeprecatedBy:        "builtin:rum.ip-mappings",
		SingleConfiguration: true,
	},
	{
		ID:                  "geo-ip-address-mappings",
		URLPath:             "/api/config/v1/geographicRegions/ipAddressMappings",
		DeprecatedBy:        "builtin:rum.ip-determination",
		SingleConfiguration: true,
	},
	{
		ID:                           "key-user-actions-mobile",
		URLPath:                      "/api/config/v1/applications/mobile/{SCOPE}/keyUserActions",
		PropertyNameOfGetAllResponse: "keyUserActions",
		SubPathAPI:                   true,
	},
}
