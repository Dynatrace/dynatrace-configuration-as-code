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
var configEndpoints = map[string]apiInput{

	"alerting-profile": {
		apiPath:            "/api/config/v1/alertingProfiles",
		deprecatedBy:       "builtin:alerting.profile",
		isNonUniqueNameApi: true,
	},
	"management-zone": {
		apiPath:      "/api/config/v1/managementZones",
		deprecatedBy: "builtin:management-zones",
	},
	"auto-tag": {
		apiPath:      "/api/config/v1/autoTags",
		deprecatedBy: "builtin:tags.auto-tagging",
	},
	"dashboard": {
		apiPath:                      "/api/config/v1/dashboards",
		propertyNameOfGetAllResponse: "dashboards",
		isNonUniqueNameApi:           true,
	},
	"notification": {
		apiPath:      "/api/config/v1/notifications",
		deprecatedBy: "builtin:problem.notifications",
	},
	"extension": {
		apiPath:                      "/api/config/v1/extensions",
		propertyNameOfGetAllResponse: "extensions",
		skipDownload:                 true,
	},
	"extension-elasticsearch": {
		apiPath:                  "/api/config/v1/extensions/dynatrace.python.elasticsearch/global",
		isSingleConfigurationApi: true,
	},
	"custom-service-java": {
		apiPath: "/api/config/v1/service/customServices/java",
	},
	"custom-service-dotnet": {
		apiPath: "/api/config/v1/service/customServices/dotNet",
	},
	"custom-service-go": {
		apiPath: "/api/config/v1/service/customServices/go",
	},
	"custom-service-nodejs": {
		apiPath: "/api/config/v1/service/customServices/nodeJS",
	},
	"custom-service-php": {
		apiPath: "/api/config/v1/service/customServices/php",
	},
	"anomaly-detection-metrics": {
		apiPath:            "/api/config/v1/anomalyDetection/metricEvents",
		deprecatedBy:       "builtin:anomaly-detection.metric-events",
		isNonUniqueNameApi: true,
	},
	// Early adopter API !
	"anomaly-detection-disks": {
		apiPath:      "/api/config/v1/anomalyDetection/diskEvents",
		deprecatedBy: "builtin:anomaly-detection.infrastructure-disks",
	},
	// Environment API not Config API
	"synthetic-location": {
		apiPath: "/api/v1/synthetic/locations",
	},
	// Environment API not Config API
	"synthetic-monitor": {
		apiPath: "/api/v1/synthetic/monitors",
	},
	"application-web": {
		apiPath: "/api/config/v1/applications/web",
	},
	"application-mobile": {
		apiPath: "/api/config/v1/applications/mobile",
	},
	"app-detection-rule": {
		apiPath:      "/api/config/v1/applicationDetectionRules",
		deprecatedBy: "builtin:rum.web.app-detection",
	},
	"aws-credentials": {
		apiPath:      "/api/config/v1/aws/credentials",
		skipDownload: true,
	},
	// Early adopter API !
	"kubernetes-credentials": {
		apiPath:      "/api/config/v1/kubernetes/credentials",
		deprecatedBy: "builtin:cloud.kubernetes",
		//isNonUniqueNameApi: true, // non-unique name handling for k8s credentials does not work, as path ID needs to be a ME-ID not a uuid; handling as unique again for now
		skipDownload: true,
	},
	"azure-credentials": {
		apiPath:      "/api/config/v1/azure/credentials",
		skipDownload: true,
	},
	"request-attributes": {
		apiPath:      "/api/config/v1/service/requestAttributes",
		deprecatedBy: "builtin:request-attributes",
	},
	"calculated-metrics-service": {
		apiPath: "/api/config/v1/calculatedMetrics/service",
	},
	// Early adopter API !
	"calculated-metrics-log": {
		apiPath: "/api/config/v1/calculatedMetrics/log",
	},
	"calculated-metrics-application-mobile": {
		apiPath: "/api/config/v1/calculatedMetrics/mobile",
	},
	"calculated-metrics-synthetic": {
		apiPath: "/api/config/v1/calculatedMetrics/synthetic",
	},
	"calculated-metrics-application-web": {
		apiPath: "/api/config/v1/calculatedMetrics/rum",
	},
	"conditional-naming-processgroup": {
		apiPath: "/api/config/v1/conditionalNaming/processGroup",
	},
	"conditional-naming-host": {
		apiPath: "/api/config/v1/conditionalNaming/host",
	},
	"conditional-naming-service": {
		apiPath: "/api/config/v1/conditionalNaming/service",
	},
	"maintenance-window": {
		apiPath:      "/api/config/v1/maintenanceWindows",
		deprecatedBy: "builtin:alerting.maintenance-window",
	},
	"request-naming-service": {
		apiPath:            "/api/config/v1/service/requestNaming",
		isNonUniqueNameApi: true,
	},
	// Environment API not Config API
	"slo": {
		apiPath:                      "/api/v2/slo",
		propertyNameOfGetAllResponse: "slo",
	},
	"credential-vault": {
		apiPath:                      "/api/config/v1/credentials",
		propertyNameOfGetAllResponse: "credentials",
		skipDownload:                 true,
	},
	"failure-detection-parametersets": {
		apiPath:      "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
		deprecatedBy: "builtin:failure-detection.environment.parameters",
	},
	"failure-detection-rules": {
		apiPath:      "/api/config/v1/service/failureDetection/parameterSelection/rules",
		deprecatedBy: "builtin:failure-detection.environment.rules",
	},
	"service-detection-full-web-request": {
		apiPath:      "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
		deprecatedBy: "builtin:service-detection.full-web-request",
	},
	"service-detection-full-web-service": {
		apiPath:      "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
		deprecatedBy: "builtin:service-detection.full-web-service",
	},
	"service-detection-opaque-web-request": {
		apiPath:      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
		deprecatedBy: "builtin:service-detection.external-web-request",
	},
	"service-detection-opaque-web-service": {
		apiPath:      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
		deprecatedBy: "builtin:service-detection.external-web-service",
	},
	// Early adopter API !
	"reports": {
		apiPath: "/api/config/v1/reports",
	},
	"frequent-issue-detection": {
		apiPath:                  "/api/config/v1/frequentIssueDetection",
		deprecatedBy:             "builtin:anomaly-detection.frequent-issues",
		isSingleConfigurationApi: true,
	},
	"data-privacy": {
		apiPath:                  "/api/config/v1/dataPrivacy",
		deprecatedBy:             "builtin:preferences.privacy",
		isSingleConfigurationApi: true,
	},
	"hosts-auto-update": {
		apiPath:                  "/api/config/v1/hosts/autoupdate",
		deprecatedBy:             "builtin:deployment.oneagent.updates",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-applications": {
		apiPath:                  "/api/config/v1/anomalyDetection/applications",
		deprecatedBy:             "builtin:anomaly-detection.rum-web, builtin:anomaly-detection.rum-mobile",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-aws": {
		apiPath:                  "/api/config/v1/anomalyDetection/aws",
		deprecatedBy:             "builtin:anomaly-detection.infrastructure-aws",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-database-services": {
		apiPath:                  "/api/config/v1/anomalyDetection/databaseServices",
		deprecatedBy:             "builtin:anomaly-detection.databases",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-hosts": {
		apiPath:                  "/api/config/v1/anomalyDetection/hosts",
		deprecatedBy:             "builtin:anomaly-detection.infrastructure-hosts",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-services": {
		apiPath:                  "/api/config/v1/anomalyDetection/services",
		deprecatedBy:             "builtin:anomaly-detection.services",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-vmware": {
		apiPath:                  "/api/config/v1/anomalyDetection/vmware",
		deprecatedBy:             "builtin:anomaly-detection.infrastructure-vmware",
		isSingleConfigurationApi: true,
	},
	"service-resource-naming": {
		apiPath:                  "/api/config/v1/service/resourceNaming",
		isSingleConfigurationApi: true,
	},
	"app-detection-rule-host": {
		apiPath:                  "/api/config/v1/applicationDetectionRules/hostDetection",
		deprecatedBy:             "builtin:rum.host-headers",
		isSingleConfigurationApi: true,
	},
	"content-resources": {
		apiPath:                  "/api/config/v1/contentResources",
		deprecatedBy:             "builtin:rum.provider-breakdown",
		isSingleConfigurationApi: true,
	},
	"allowed-beacon-origins": {
		apiPath:                  "/api/config/v1/allowedBeaconOriginsForCors",
		deprecatedBy:             "builtin:rum.web.beacon-domain-origins",
		isSingleConfigurationApi: true,
	},
	"geo-ip-detection-headers": {
		apiPath:                  "/api/config/v1/geographicRegions/ipDetectionHeaders",
		deprecatedBy:             "builtin:rum.ip-mappings",
		isSingleConfigurationApi: true,
	},
	"geo-ip-address-mappings": {
		apiPath:                  "/api/config/v1/geographicRegions/ipAddressMappings",
		deprecatedBy:             "builtin:rum.ip-determination",
		isSingleConfigurationApi: true,
	},
}
