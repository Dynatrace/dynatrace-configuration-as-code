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
}

var settings20ApiMap = map[string]apiInputSettings20{

	"builtin:span-entry-points": {
		namePropertyJsonPath: "$.entryPointRule.ruleName",
	},
	"builtin:tokens.token-settings": {
		unique: true,
	},
	"builtin:problem.notifications": {
		namePropertyJsonPath: "$.displayName",
	},
	"builtin:logmonitoring.log-events": {
		namePropertyJsonPath: "$.summary",
	},
}

// Settings20SchemaApi is the API used internally to get all the settings 2.0 schemas
var Settings20SchemaApi Api = &apiImpl{
	id:                           "schemas",
	apiPath:                      "/api/v2/settings/schemas",
	propertyNameOfGetAllResponse: "items",
}

// Settings20ObjectsApi is the API used internally to get all the settings 2.0 objects
var Settings20ObjectsApi Api = &apiImpl{
	id:                           "objects",
	apiPath:                      "/api/v2/settings/objects",
	propertyNameOfGetAllResponse: "items",
}

var standardApiPropertyNameOfGetAllResponse = "values"

type Api interface {
	GetUrl(environment environment.Environment) string
	GetUrlFromEnvironmentUrl(environmentUrl string) string
	GetId() string
	GetApiPath() string
	GetPropertyNameOfGetAllResponse() string
	IsStandardApi() bool
	IsSettings20Api() bool
}

type apiInputSettings20 struct {
	namePropertyJsonPath string
	unique               bool
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

type settings20Impl struct {
	id                   string
	apiPath              string
	namePropertyJsonPath string
	unique               bool
}

func NewApis() map[string]Api {

	apis := make(map[string]Api)

	for id, details := range apiMap {
		apis[id] = newApi(id, details)
	}

	for id, details := range settings20ApiMap {
		apis[id] = &settings20Impl{
			id:                   id,
			apiPath:              id,
			namePropertyJsonPath: details.namePropertyJsonPath,
			unique:               details.namePropertyJsonPath == "",
		}
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

func (a *apiImpl) IsSettings20Api() bool {
	return false
}

func (a *settings20Impl) GetUrl(environment environment.Environment) string {
	return environment.GetEnvironmentUrl() + "settings/objects"
}

func (a *settings20Impl) GetUrlFromEnvironmentUrl(environmentUrl string) string {
	return environmentUrl + "/api/v2/settings/objects"
}

func (a *settings20Impl) GetId() string {
	return a.id
}

func (a *settings20Impl) GetApiPath() string {
	return a.apiPath
}

func (a *settings20Impl) GetPropertyNameOfGetAllResponse() string {
	return a.namePropertyJsonPath
}

func (a *settings20Impl) IsStandardApi() bool {
	return false
}

func (a *settings20Impl) IsSettings20Api() bool {
	return true
}

func IsApi(dir string) bool {
	_, okClassicApi := apiMap[dir]
	_, okSettings20Api := settings20ApiMap[dir]

	return okClassicApi || okSettings20Api
}

// tests if part of project folder path contains an API
// folders with API in path are not valid projects
func ContainsApiName(path string) bool {
	for api := range apiMap {
		if strings.Contains(path, api) {
			return true
		}
	}
	for api := range settings20ApiMap {
		if strings.Contains(path, api) {
			return true
		}
	}
	return false
}
