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

import (
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
)

//go:generate mockgen -source=api.go -destination=api_mock.go -package=api Api

var apiMap = map[string]apiInput{

	// Early adopter API !
	"alerting-profile": {
		apiPath: "/api/config/v1/alertingProfiles",
	},
	"management-zone": {
		apiPath: "/api/config/v1/managementZones",
	},
	"auto-tag": {
		apiPath: "/api/config/v1/autoTags",
	},
	// Early adopter API !
	"dashboard": {
		apiPath:                      "/api/config/v1/dashboards",
		propertyNameOfGetAllResponse: "dashboards",
	},
	"notification": {
		apiPath: "/api/config/v1/notifications",
	},
	"extension": {
		apiPath:                      "/api/config/v1/extensions",
		propertyNameOfGetAllResponse: "extensions",
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
	// Early adopter API !
	"anomaly-detection-metrics": {
		apiPath: "/api/config/v1/anomalyDetection/metricEvents",
	},
	// Early adopter API !
	// Environment API not Config API
	"synthetic-location": {
		apiPath: "/api/v1/synthetic/locations",
	},
	// Early adopter API !
	// Environment API not Config API
	"synthetic-monitor": {
		apiPath: "/api/v1/synthetic/monitors",
	},
	"application": {
		apiPath: "/api/config/v1/applications/web",
	},
	"application-web": {
		apiPath: "/api/config/v1/applications/web",
	},
	"application-mobile": {
		apiPath: "/api/config/v1/applications/mobile",
	},
	"app-detection-rule": {
		apiPath: "/api/config/v1/applicationDetectionRules",
	},
	"aws-credentials": {
		apiPath: "/api/config/v1/aws/credentials",
	},
	// Early adopter API !
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
	// Early adopter API !
	"calculated-metrics-log": {
		apiPath: "/api/config/v1/calculatedMetrics/log",
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
		apiPath: "/api/config/v1/service/requestNaming",
	},

	// Early adopter API !
	// Environment API not Config API
	"slo": {
		apiPath:                      "/api/v2/slo",
		propertyNameOfGetAllResponse: "slo",
	},

	// Early adopter API !
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

	"service-detection-web-service": {
		apiPath: "/api/config/v1/service/detectionRules/FULL_WEB_SERVICE",
	},

	"service-detection-web-request": {
		apiPath: "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST",
	},
}

var standardApiPropertyNameOfGetAllResponse = "values"

type Api interface {
	GetUrl(environment environment.Environment) string
	GetUrlFromEnvironmentUrl(environmentUrl string) string
	GetId() string
	GetApiPath() string
	GetPropertyNameOfGetAllResponse() string
	IsStandardApi() bool
}

type apiInput struct {
	apiPath                      string
	propertyNameOfGetAllResponse string
}

type apiImpl struct {
	id                           string
	apiPath                      string
	propertyNameOfGetAllResponse string
}

func NewApis() map[string]Api {

	apis := make(map[string]Api)

	for id, details := range apiMap {
		apis[id] = newApi(id, details)
	}

	return apis
}

func newApi(id string, input apiInput) Api {
	if input.propertyNameOfGetAllResponse == "" {
		return NewStandardApi(id, input.apiPath)
	}
	return NewApi(id, input.apiPath, input.propertyNameOfGetAllResponse)
}

// NewStandardApi creates an API with propertyNameOfGetAllResponse set to "values"
func NewStandardApi(id string, apiPath string) Api {
	return NewApi(id, apiPath, standardApiPropertyNameOfGetAllResponse)
}

func NewApi(id string, apiPath string, propertyNameOfGetAllResponse string) Api {

	// TODO log warning if the user tries to create an API with a id not present in map above
	// This means that a user runs monaco with an untested api

	return &apiImpl{
		id:                           id,
		apiPath:                      apiPath,
		propertyNameOfGetAllResponse: propertyNameOfGetAllResponse,
	}
}

func (a *apiImpl) GetUrl(environment environment.Environment) string {
	return environment.GetEnvironmentUrl() + a.apiPath
}

func (a *apiImpl) GetUrlFromEnvironmentUrl(environmentUrl string) string {
	return environmentUrl + a.apiPath
}

func (a *apiImpl) GetId() string {
	return a.id
}

func (a *apiImpl) GetApiPath() string {
	return a.apiPath
}

func (a *apiImpl) GetPropertyNameOfGetAllResponse() string {
	return a.propertyNameOfGetAllResponse
}

func (a *apiImpl) IsStandardApi() bool {
	return a.propertyNameOfGetAllResponse == standardApiPropertyNameOfGetAllResponse
}

func IsApi(dir string) bool {
	_, ok := apiMap[dir]
	return ok
}

// tests if part of project folder path contains an API
// folders with API in path are not valid projects
func ContainsApiName(path string) bool {
	for api := range apiMap {
		if strings.Contains(path, api) {
			return true
		}
	}
	return false
}
