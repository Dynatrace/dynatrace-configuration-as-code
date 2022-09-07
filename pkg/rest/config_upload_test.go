//go:build unit
// +build unit

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

package rest

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"gotest.tools/assert"
)

var testReportsApi = api.NewStandardApi("reports", "/api/config/v1/reports", false, "")
var testDashboardApi = api.NewStandardApi("dashboard", "/api/config/v1/dashboards", true, "dashboard-v2")
var testMobileAppApi = api.NewStandardApi("application-mobile", "/api/config/v1/applications/mobile", false, "")
var testServiceDetectionApi = api.NewStandardApi("service-detection-full-web-request", "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST", false, "")
var testSyntheticApi = api.NewStandardApi("synthetic-monitor", "/api/environment/v1/synthetic/monitor", false, "")

func TestTranslateGenericValuesOnStandardResponse(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(response, "extensions")

	assert.NilError(t, err)
	assert.Check(t, len(values) == 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "bar")
}

func TestTranslateGenericValuesOnIdMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	_, err := translateGenericValues(response, "extensions")

	assert.ErrorContains(t, err, "config of type extensions was invalid: No id")
}

func TestTranslateGenericValuesOnNameMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(response, "extensions")

	assert.NilError(t, err)
	assert.Check(t, len(values) == 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "foo")
}

func TestTranslateGenericValuesForReportsEndpoint(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"
	entry["dashboardId"] = "dashboardId"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(response, "reports")

	assert.NilError(t, err)
	assert.Check(t, len(values) == 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "dashboardId")
}

func TestJoinUrl(t *testing.T) {
	urlBase := "url/"
	path := "path"

	joinedUrl := joinUrl(urlBase, path)
	assert.Equal(t, joinedUrl, "url/path")

	urlBase = "url/"
	path = "path"

	joinedUrl = joinUrl(urlBase, path)
	assert.Equal(t, joinedUrl, "url/path")

	urlBase = "url"
	path = "path"

	joinedUrl = joinUrl(urlBase, path)
	assert.Equal(t, joinedUrl, "url/path")

	urlBase = "url"
	path = " "

	joinedUrl = joinUrl(urlBase, path)
	assert.Equal(t, joinedUrl, "url")

	urlBase = "url/"
	path = " "

	joinedUrl = joinUrl(urlBase, path)
	assert.Equal(t, joinedUrl, "url")
}

func TestIsReportsApi(t *testing.T) {
	isTrue := isReportsApi(testReportsApi)
	assert.Equal(t, true, isTrue)

	isFalse := isReportsApi(testDashboardApi)
	assert.Equal(t, false, isFalse)
}

func TestIsAnyApplicationApi(t *testing.T) {

	assert.Equal(t, true, isAnyApplicationApi(testMobileAppApi))

	testWebApi := api.NewStandardApi("application-web", "/api/config/v1/applications/web", false, "")
	assert.Equal(t, true, isAnyApplicationApi(testWebApi))

	assert.Equal(t, false, isAnyApplicationApi(testDashboardApi))
}

func TestIsMobileApp(t *testing.T) {
	isTrue := isMobileApp(testMobileAppApi)
	assert.Equal(t, true, isTrue)

	isFalse := isMobileApp(testDashboardApi)
	assert.Equal(t, false, isFalse)
}

func TestIsAnyServiceDetectionApi(t *testing.T) {
	isTrue := isAnyServiceDetectionApi(testServiceDetectionApi)
	assert.Equal(t, true, isTrue)

	isFalse := isAnyServiceDetectionApi(testDashboardApi)
	assert.Equal(t, false, isFalse)
}

func TestIsApiDashboard(t *testing.T) {
	isTrue := isApiDashboard(testDashboardApi)
	assert.Equal(t, true, isTrue)

	isFalse := isApiDashboard(testReportsApi)
	assert.Equal(t, false, isFalse)
}

func Test_success(t *testing.T) {
	tests := []struct {
		name string
		resp Response
		want bool
	}{
		{
			"200 is success",
			Response{
				StatusCode: 200,
			},
			true,
		},
		{
			"201 is success",
			Response{
				StatusCode: 201,
			},
			true,
		},
		{
			"401 is NOT success",
			Response{
				StatusCode: 401,
			},
			false,
		},
		{
			"503 is NOT success",
			Response{
				StatusCode: 503,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := success(tt.resp); got != tt.want {
				t.Errorf("success() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isServerError(t *testing.T) {
	tests := []struct {
		name string
		resp Response
		want bool
	}{
		{
			"200 is NOT server error",
			Response{
				StatusCode: 200,
			},
			false,
		},
		{
			"201 is NOT server error",
			Response{
				StatusCode: 201,
			},
			false,
		},
		{
			"401 is NOT server error",
			Response{
				StatusCode: 401,
			},
			false,
		},
		{
			"503 is server error",
			Response{
				StatusCode: 503,
			},
			true,
		},
		{
			"500 is server error",
			Response{
				StatusCode: 500,
			},
			true,
		},
		{
			"greater than 599 is NOT server error",
			Response{
				StatusCode: 600,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isServerError(tt.resp); got != tt.want {
				t.Errorf("isServerError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isApplicationNotReadyYet(t *testing.T) {
	type args struct {
		resp   Response
		theApi api.Api
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Server Error on synthetic counted as app not ready (issue in error reporting for unknown App IDs in some Dynatrace versions)",
			args{
				Response{
					StatusCode: 500,
					Body:       nil,
					Headers:    nil,
				},
				testSyntheticApi,
			},
			true,
		},
		{
			"Server Error on application API counts as not ready (can happen on update)",
			args{
				Response{
					StatusCode: 503,
					Body:       nil,
					Headers:    nil,
				},
				testMobileAppApi,
			},
			true,
		},
		{
			"Server Error on unexpected API not counted as App not ready",
			args{
				Response{
					StatusCode: 503,
					Body:       nil,
					Headers:    nil,
				},
				testDashboardApi,
			},
			false,
		},
		{
			"User error response of 'Unknown Application' counted as not ready (can happen if App was just created)",
			args{
				Response{
					StatusCode: 400,
					Body:       []byte("Unknown application(s) APP-422142"),
					Headers:    nil,
				},
				testMobileAppApi,
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isApplicationNotReadyYet(tt.args.resp, tt.args.theApi); got != tt.want {
				t.Errorf("isApplicationNotReadyYet() = %v, want %v", got, tt.want)
			}
		})
	}
}
