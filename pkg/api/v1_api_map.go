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

// v1ApiMap contains API definitions present in v1 to allow conversion and fallback deployment of v1
// This includes deprecated APIs removed with v2, as well as the '-v2' non-unique-name APIs moved to being the default
// and dropping the '-v2' suffix with v2.
var v1ApiMap = map[string]apiInput{

	"alerting-profile": {
		apiPath: "/api/config/v1/alertingProfiles",
	},
	"management-zone": {
		apiPath: "/api/config/v1/managementZones",
	},
	"auto-tag": {
		apiPath: "/api/config/v1/autoTags",
	},
	"dashboard": {
		apiPath:                      "/api/config/v1/dashboards",
		propertyNameOfGetAllResponse: "dashboards",
		isDeprecatedBy:               "dashboard-v2",
	},
	"dashboard-v2": {
		apiPath:                      "/api/config/v1/dashboards",
		propertyNameOfGetAllResponse: "dashboards",
		isNonUniqueNameApi:           true,
	},
	"notification": {
		apiPath: "/api/config/v1/notifications",
	},
	"extension": {
		apiPath:                      "/api/config/v1/extensions",
		propertyNameOfGetAllResponse: "extensions",
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
		apiPath: "/api/config/v1/anomalyDetection/metricEvents",
	},
	"anomaly-detection-disks": {
		apiPath: "/api/config/v1/anomalyDetection/diskEvents",
	},
	"synthetic-location": {
		apiPath: "/api/v1/synthetic/locations",
	},
	"synthetic-monitor": {
		apiPath: "/api/v1/synthetic/monitors",
	},
	"application": {
		apiPath:        "/api/config/v1/applications/web",
		isDeprecatedBy: "application-web",
	},
	"application-web": {
		apiPath: "/api/config/v1/applications/web",
	},
	"application-mobile": {
		apiPath: "/api/config/v1/applications/mobile",
	},
	"app-detection-rule": {
		apiPath:        "/api/config/v1/applicationDetectionRules",
		isDeprecatedBy: "app-detection-rule-v2",
	},
	"app-detection-rule-v2": {
		apiPath:            "/api/config/v1/applicationDetectionRules",
		isNonUniqueNameApi: true,
	},
	"aws-credentials": {
		apiPath: "/api/config/v1/aws/credentials",
	},
	"kubernetes-credentials": {
		apiPath: "/api/config/v1/kubernetes/credentials",
	},
	"azure-credentials": {
		apiPath: "/api/config/v1/azure/credentials",
	},
	"request-attributes": {
		apiPath: "/api/config/v1/service/requestAttributes",
	},
	"calculated-metrics-service": {
		apiPath: "/api/config/v1/calculatedMetrics/service",
	},
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
		apiPath: "/api/config/v1/maintenanceWindows",
	},
	"request-naming-service": {
		apiPath:        "/api/config/v1/service/requestNaming",
		isDeprecatedBy: "request-naming-service-v2",
	},
	"request-naming-service-v2": {
		apiPath:            "/api/config/v1/service/requestNaming",
		isNonUniqueNameApi: true,
	},
	"slo": {
		apiPath:                      "/api/v2/slo",
		propertyNameOfGetAllResponse: "slo",
	},
	"credential-vault": {
		apiPath:                      "/api/config/v1/credentials",
		propertyNameOfGetAllResponse: "credentials",
	},
	"failure-detection-parametersets": {
		apiPath: "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
	},
	"failure-detection-rules": {
		apiPath: "/api/config/v1/service/failureDetection/parameterSelection/rules",
	},
	"service-detection-full-web-request": {
		apiPath: "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
	},
	"service-detection-full-web-service": {
		apiPath: "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
	},
	"service-detection-opaque-web-request": {
		apiPath: "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
	},
	"service-detection-opaque-web-service": {
		apiPath: "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
	},
	"reports": {
		apiPath: "/api/config/v1/reports",
	},
	"frequent-issue-detection": {
		apiPath:                  "/api/config/v1/frequentIssueDetection",
		isSingleConfigurationApi: true,
	},
	"data-privacy": {
		apiPath:                  "/api/config/v1/dataPrivacy",
		isSingleConfigurationApi: true,
	},
	"hosts-auto-update": {
		apiPath:                  "/api/config/v1/hosts/autoupdate",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-applications": {
		apiPath:                  "/api/config/v1/anomalyDetection/applications",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-aws": {
		apiPath:                  "/api/config/v1/anomalyDetection/aws",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-database-services": {
		apiPath:                  "/api/config/v1/anomalyDetection/databaseServices",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-hosts": {
		apiPath:                  "/api/config/v1/anomalyDetection/hosts",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-services": {
		apiPath:                  "/api/config/v1/anomalyDetection/services",
		isSingleConfigurationApi: true,
	},
	"anomaly-detection-vmware": {
		apiPath:                  "/api/config/v1/anomalyDetection/vmware",
		isSingleConfigurationApi: true,
	},
	"service-resource-naming": {
		apiPath:                  "/api/config/v1/service/resourceNaming",
		isSingleConfigurationApi: true,
	},
	"app-detection-rule-host": {
		apiPath:                  "/api/config/v1/applicationDetectionRules/hostDetection",
		isSingleConfigurationApi: true,
	},
	"content-resources": {
		apiPath:                  "/api/config/v1/contentResources",
		isSingleConfigurationApi: true,
	},
	"allowed-beacon-origins": {
		apiPath:                  "/api/config/v1/allowedBeaconOriginsForCors",
		isSingleConfigurationApi: true,
	},
	"geo-ip-detection-headers": {
		apiPath:                  "/api/config/v1/geographicRegions/ipDetectionHeaders",
		isSingleConfigurationApi: true,
	},
	"geo-ip-address-mappings": {
		apiPath:                  "/api/config/v1/geographicRegions/ipAddressMappings",
		isSingleConfigurationApi: true,
	},
}

// GetV2ApiId returns the ID of APIs in v2 - replacing deprecated APIs with their new version and dropping the -v2 marker
// from APIs introducing the breaking change of handling non-unique-names. This is used in v1 -> v2 conversion
func GetV2ApiId(forV1Api Api) string {
	currentApiId := forV1Api.GetId()

	if forV1Api.IsDeprecatedApi() {
		currentApiId = forV1Api.IsDeprecatedBy()
	}

	return strings.TrimSuffix(currentApiId, "-v2")
}
