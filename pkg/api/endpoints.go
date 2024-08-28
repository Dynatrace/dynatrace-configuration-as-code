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

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"sync"
	"time"
)

const (
	AlertingProfile                      = "alerting-profile"
	NetworkZone                          = "network-zone"
	ManagementZone                       = "management-zone"
	Autotag                              = "auto-tag"
	Dashboard                            = "dashboard"
	DashboardShareSettings               = "dashboard-share-settings"
	DashboardV2                          = "dashboard-v2"
	Notification                         = "notification"
	Extension                            = "extension"
	ExtensionElasticSearch               = "extension-elasticsearch"
	CustomServiceJava                    = "custom-service-java"
	CustomServiceDotNet                  = "custom-service-dotnet"
	CustomServiceGo                      = "custom-service-go"
	CustomServiceNodeJs                  = "custom-service-nodejs"
	CustomServicePhp                     = "custom-service-php"
	AnomalyDetectionMetrics              = "anomaly-detection-metrics"
	AnomalyDetectionDisks                = "anomaly-detection-disks"
	SyntheticLocation                    = "synthetic-location"
	SyntheticMonitor                     = "synthetic-monitor"
	ApplicationWeb                       = "application-web"
	ApplicationMobile                    = "application-mobile"
	AppDetectionRule                     = "app-detection-rule"
	AwsCredentials                       = "aws-credentials"
	KubernetesCredentials                = "kubernetes-credentials" // #nosec G101
	AzureCredentials                     = "azure-credentials"      // #nosec G101
	RequestAttributes                    = "request-attributes"
	CalculatedMetricsService             = "calculated-metrics-service"
	CalculatedMetricsLog                 = "calculated-metrics-log"
	CalculatedMetricsApplicationMobile   = "calculated-metrics-application-mobile"
	CalculatedMetricsSynthetic           = "calculated-metrics-synthetic"
	CalculatedMetricsApplicationWeb      = "calculated-metrics-application-web"
	ConditionalNamingProcessgroup        = "conditional-naming-processgroup"
	ConditionalNamingHost                = "conditional-naming-host"
	ConditionalNamingService             = "conditional-naming-service"
	MaintenanceWindow                    = "maintenance-window"
	RequestNamingService                 = "request-naming-service"
	Slo                                  = "slo"
	CredentialVault                      = "credential-vault" // #nosec G101
	FailureDetectionParametersets        = "failure-detection-parametersets"
	FailureDetectionRules                = "failure-detection-rules"
	ServiceDetectionFullWebRequest       = "service-detection-full-web-request"
	ServiceDetectionFullWebService       = "service-detection-full-web-service"
	ServiceDetectionOpaqueWebRequest     = "service-detection-opaque-web-request"
	ServiceDetectionOpaqueWebService     = "service-detection-opaque-web-service"
	Reports                              = "reports"
	FrequentIssueDetection               = "frequent-issue-detection"
	DataPrivacy                          = "data-privacy"
	HostsAutoUpdate                      = "hosts-auto-update"
	AnomalyDetectionApplications         = "anomaly-detection-applications"
	AnomalyDetectionAws                  = "anomaly-detection-aws"
	AnomalyDetectionDatabaseServices     = "anomaly-detection-database-services"
	AnomalyDetectionHosts                = "anomaly-detection-hosts"
	AnomalyDetectionServices             = "anomaly-detection-services"
	AnomalyDetectionVmware               = "anomaly-detection-vmware"
	ServiceResourceNaming                = "service-resource-naming"
	AppDetectionRuleHost                 = "app-detection-rule-host"
	ContentResources                     = "content-resources"
	AllowedBeaconOrigins                 = "allowed-beacon-origins"
	GeoIpDetectionHeaders                = "geo-ip-detection-headers"
	GeoIpAddressMappings                 = "geo-ip-address-mappings"
	KeyUserActionsMobile                 = "key-user-actions-mobile"
	KeyUserActionsWeb                    = "key-user-actions-web"
	UserActionAndSessionPropertiesMobile = "user-action-and-session-properties-mobile"
)

func removeURLsFromPublicAccess(m map[string]any) {
	if publicAccess, found := m["publicAccess"]; found {
		publicAccessMap := publicAccess.(map[string]any)
		delete(publicAccessMap, "urls")
	}
}

var configEndpoints []API = nil
var configEndpointsOnce sync.Once

// NewAPIs returns collection of predefined API to work with Dynatrace
func NewAPIs() APIs {

	configEndpointsOnce.Do(func() {
		// Dashboard has DashboardShareSettings as child API and so is defined here explicitly
		var dashboardAPI = API{
			ID:                           Dashboard,
			URLPath:                      "/api/config/v1/dashboards",
			PropertyNameOfGetAllResponse: "dashboards",
			NonUniqueName:                true,
		}

		// ApplicationWeb has KeyUserActionsWeb as a child API and so is defined here explicitly
		var applicationWebAPI = API{
			ID:                           ApplicationWeb,
			URLPath:                      "/api/config/v1/applications/web",
			PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		}

		// ApplicationMobile has KeyUserActionsMobile and UserActionAndSessionPropertiesMobile as child APIs and so is defined here explicitly
		var applicationMobileAPI = API{
			ID:                           ApplicationMobile,
			URLPath:                      "/api/config/v1/applications/mobile",
			PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
		}

		// configEndpoints is map of the http endpoints for configuration API (aka classic/config endpoints).
		configEndpoints = []API{
			{
				ID:                           AlertingProfile,
				URLPath:                      "/api/config/v1/alertingProfiles",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:alerting.profile",
				NonUniqueName:                true,
			},
			{
				ID:                           NetworkZone,
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
				ID:                           ManagementZone,
				URLPath:                      "/api/config/v1/managementZones",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:management-zones",
			},
			{
				ID:                           Autotag,
				URLPath:                      "/api/config/v1/autoTags",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:tags.auto-tagging",
			},
			dashboardAPI,
			{
				ID:                  DashboardShareSettings,
				URLPath:             "/api/config/v1/dashboards/{SCOPE}/shareSettings",
				Parent:              &dashboardAPI,
				SingleConfiguration: true,
				NonDeletable:        true,
				TweakResponseFunc:   removeURLsFromPublicAccess,
			},
			{
				ID:                           Notification,
				URLPath:                      "/api/config/v1/notifications",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:problem.notifications",
			},
			{
				ID:                           Extension,
				URLPath:                      "/api/config/v1/extensions",
				PropertyNameOfGetAllResponse: "extensions",
				SkipDownload:                 true,
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
				DeprecatedBy:                 "builtin:anomaly-detection.metric-events",
				NonUniqueName:                true,
			},
			// Early adopter API !
			{
				ID:                           AnomalyDetectionDisks,
				URLPath:                      "/api/config/v1/anomalyDetection/diskEvents",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:anomaly-detection.infrastructure-disks",
			},
			// Environment API not Config API
			{
				ID:                           SyntheticLocation,
				URLPath:                      "/api/v1/synthetic/locations",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
			},
			// Environment API not Config API
			{
				ID:                           SyntheticMonitor,
				URLPath:                      "/api/v1/synthetic/monitors",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
			},
			applicationWebAPI,
			applicationMobileAPI,
			{
				ID:                           AppDetectionRule,
				URLPath:                      "/api/config/v1/applicationDetectionRules",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:rum.web.app-detection",
			},
			{
				ID:                           AwsCredentials,
				URLPath:                      "/api/config/v1/aws/credentials",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				SkipDownload:                 true,
			},
			// Early adopter API !
			{
				ID:                           KubernetesCredentials,
				URLPath:                      "/api/config/v1/kubernetes/credentials",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:cloud.kubernetes",
				//NonUniqueName: true, // non-unique name handling for k8s credentials does not work, as path ID needs to be a ME-ID not a uuid; handling as unique again for now
				SkipDownload: true,
			},
			{
				ID:                           AzureCredentials,
				URLPath:                      "/api/config/v1/azure/credentials",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				SkipDownload:                 true,
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
				DeprecatedBy:                 "builtin:alerting.maintenance-window",
			},
			{
				ID:                           RequestNamingService,
				URLPath:                      "/api/config/v1/service/requestNaming",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				NonUniqueName:                true,
			},
			// Environment API not Config API
			{
				ID:                           Slo,
				URLPath:                      "/api/v2/slo",
				PropertyNameOfGetAllResponse: "slo",
			},
			{
				ID:                           CredentialVault,
				URLPath:                      "/api/config/v1/credentials",
				PropertyNameOfGetAllResponse: "credentials",
				SkipDownload:                 true,
			},
			{
				ID:                           FailureDetectionParametersets,
				URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/parameterSets",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:failure-detection.environment.parameters",
			},
			{
				ID:                           FailureDetectionRules,
				URLPath:                      "/api/config/v1/service/failureDetection/parameterSelection/rules",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:failure-detection.environment.rules",
			},
			{
				ID:                           ServiceDetectionFullWebRequest,
				URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:service-detection.full-web-request",
			},
			{
				ID:                           ServiceDetectionFullWebService,
				URLPath:                      "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:service-detection.full-web-service",
			},
			{
				ID:                           ServiceDetectionOpaqueWebRequest,
				URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_REQUEST",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:service-detection.external-web-request",
			},
			{
				ID:                           ServiceDetectionOpaqueWebService,
				URLPath:                      "/api/config/v1/service/detectionRules/OPAQUE_AND_EXTERNAL_WEB_SERVICE",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
				DeprecatedBy:                 "builtin:service-detection.external-web-service",
			},
			// Early adopter API !
			{
				ID:                           Reports,
				URLPath:                      "/api/config/v1/reports",
				PropertyNameOfGetAllResponse: StandardApiPropertyNameOfGetAllResponse,
			},
			{
				ID:                  FrequentIssueDetection,
				URLPath:             "/api/config/v1/frequentIssueDetection",
				DeprecatedBy:        "builtin:anomaly-detection.frequent-issues",
				SingleConfiguration: true,
			},
			{
				ID:                  DataPrivacy,
				URLPath:             "/api/config/v1/dataPrivacy",
				DeprecatedBy:        "builtin:preferences.privacy",
				SingleConfiguration: true,
			},
			{
				ID:                  HostsAutoUpdate,
				URLPath:             "/api/config/v1/hosts/autoupdate",
				DeprecatedBy:        "builtin:deployment.oneagent.updates",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionApplications,
				URLPath:             "/api/config/v1/anomalyDetection/applications",
				DeprecatedBy:        "builtin:anomaly-detection.rum-web, builtin:anomaly-detection.rum-mobile",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionAws,
				URLPath:             "/api/config/v1/anomalyDetection/aws",
				DeprecatedBy:        "builtin:anomaly-detection.infrastructure-aws",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionDatabaseServices,
				URLPath:             "/api/config/v1/anomalyDetection/databaseServices",
				DeprecatedBy:        "builtin:anomaly-detection.databases",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionHosts,
				URLPath:             "/api/config/v1/anomalyDetection/hosts",
				DeprecatedBy:        "builtin:anomaly-detection.infrastructure-hosts",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionServices,
				URLPath:             "/api/config/v1/anomalyDetection/services",
				DeprecatedBy:        "builtin:anomaly-detection.services",
				SingleConfiguration: true,
			},
			{
				ID:                  AnomalyDetectionVmware,
				URLPath:             "/api/config/v1/anomalyDetection/vmware",
				DeprecatedBy:        "builtin:anomaly-detection.infrastructure-vmware",
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
				DeprecatedBy:        "builtin:rum.host-headers",
				SingleConfiguration: true,
			},
			{
				ID:                  ContentResources,
				URLPath:             "/api/config/v1/contentResources",
				DeprecatedBy:        "builtin:rum.provider-breakdown",
				SingleConfiguration: true,
			},
			{
				ID:                  AllowedBeaconOrigins,
				URLPath:             "/api/config/v1/allowedBeaconOriginsForCors",
				DeprecatedBy:        "builtin:rum.web.beacon-domain-origins",
				SingleConfiguration: true,
			},
			{
				ID:                  GeoIpDetectionHeaders,
				URLPath:             "/api/config/v1/geographicRegions/ipDetectionHeaders",
				DeprecatedBy:        "builtin:rum.ip-mappings",
				SingleConfiguration: true,
			},
			{
				ID:                  GeoIpAddressMappings,
				URLPath:             "/api/config/v1/geographicRegions/ipAddressMappings",
				DeprecatedBy:        "builtin:rum.ip-determination",
				SingleConfiguration: true,
			},
			{
				ID:                           KeyUserActionsMobile,
				URLPath:                      "/api/config/v1/applications/mobile/{SCOPE}/keyUserActions",
				PropertyNameOfGetAllResponse: "keyUserActions",
				PropertyNameOfIdentifier:     "name",
				Parent:                       &applicationMobileAPI,
			},
			{
				ID:                           KeyUserActionsWeb,
				URLPath:                      "/api/config/v1/applications/web/{SCOPE}/keyUserActions",
				PropertyNameOfGetAllResponse: "keyUserActionList",
				Parent:                       &applicationWebAPI,
				TweakResponseFunc:            func(m map[string]any) { delete(m, "meIdentifier") },
				CheckEqualFunc: func(existing map[string]any, current map[string]any) bool {
					return existing["name"] == current["name"] &&
						existing["actionType"] == current["actionType"] &&
						existing["domain"] == current["domain"]
				},
				DeployWaitDuration: time.Duration(environment.GetEnvValueIntLog(environment.KeyUserActionWebWaitSecondsEnvKey)) * time.Second,
			},
			{
				ID:      UserActionAndSessionPropertiesMobile,
				URLPath: "/api/config/v1/applications/mobile/{SCOPE}/userActionAndSessionProperties",
				Parent:  &applicationMobileAPI,
			},
		}
	})

	return newAPIs(configEndpoints)
}
