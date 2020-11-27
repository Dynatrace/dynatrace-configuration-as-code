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

var apiMap = map[string]string{
	// Early adopter API !
	"alerting-profile": "/api/config/v1/alertingProfiles",
	"management-zone":  "/api/config/v1/managementZones",
	"auto-tag":         "/api/config/v1/autoTags",
	// Early adopter API !
	"dashboard":           "/api/config/v1/dashboards",
	"notification":        "/api/config/v1/notifications",
	"extension":           "/api/config/v1/extensions",
	"custom-service-java": "/api/config/v1/service/customServices/java",
	// Early adopter API !
	"anomaly-detection-metrics": "/api/config/v1/anomalyDetection/metricEvents",
	// Early adopter API !
	// Environment API not Config API
	"synthetic-location": "/api/v1/synthetic/locations",
	// Early adopter API !
	// Environment API not Config API
	"synthetic-monitor":  "/api/v1/synthetic/monitors",
	"application":        "/api/config/v1/applications/web",
	"app-detection-rule": "/api/config/v1/applicationDetectionRules",
	"aws-credentials":    "/api/config/v1/aws/credentials",
	// Early adopter API !
	"kubernetes-credentials": "/api/config/v1/kubernetes/credentials",
	"azure-credentials":      "/api/config/v1/azure/credentials",

	"request-attributes": "/api/config/v1/service/requestAttributes",

	"calculated-metrics-service": "/api/config/v1/calculatedMetrics/service",
	// Early adopter API !
	"calculated-metrics-log": "/api/config/v1/calculatedMetrics/log",

	"conditional-naming-processgroup": "/api/config/v1/conditionalNaming/processGroup",
	"conditional-naming-host":         "/api/config/v1/conditionalNaming/host",
	"conditional-naming-service":      "/api/config/v1/conditionalNaming/service",
	"maintenance-window":              "/api/config/v1/maintenanceWindows",
    "request-naming":                  "/api/config/v1/service/requestNaming",
}

type Api interface {
	GetUrl(environment environment.Environment) string
	GetId() string
}

type apiImpl struct {
	id      string
	apiPath string
}

func NewApis() map[string]Api {

	apis := make(map[string]Api)

	for id, details := range apiMap {
		apis[id] = newApi(id, details)
	}

	return apis
}

func newApi(id string, apiPath string) Api {

	return NewApi(id, apiPath)
}

func NewApi(id string, apiPath string) Api {
	return &apiImpl{
		id:      id,
		apiPath: apiPath,
	}
}

func (a *apiImpl) GetUrl(environment environment.Environment) string {
	return environment.GetEnvironmentUrl() + a.apiPath
}

func (a *apiImpl) GetId() string {
	return a.id
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
