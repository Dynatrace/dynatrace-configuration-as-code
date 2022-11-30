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
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
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

func Test_retryReturnsFirstSuccessfulResponse(t *testing.T) {
	i := 0
	mockCall := sendingRequest(func(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
		if i < 3 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	gotResp, err := retry(nil, mockCall, "dont matter", "some/path", []byte("body"), "token", 5, 1)
	assert.NilError(t, err)
	assert.Equal(t, gotResp.StatusCode, 200)
	assert.Equal(t, string(gotResp.Body), "Success")
}

func Test_retryFailsAfterDefinedTries(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := sendingRequest(func(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := retry(nil, mockCall, "dont matter", "some/path", []byte("body"), "token", maxRetries, 1)
	assert.Check(t, err != nil)
	assert.Check(t, i == 2)
}

func Test_retryReturnContainsOriginalApiError(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := sendingRequest(func(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{}, fmt.Errorf("Something wrong")
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := retry(nil, mockCall, "dont matter", "some/path", []byte("body"), "token", maxRetries, 1)
	assert.Check(t, err != nil)
	assert.ErrorContains(t, err, "Something wrong")
}

func Test_retryReturnContainsHttpErrorIfNotSuccess(t *testing.T) {
	maxRetries := 2
	i := 0
	mockCall := sendingRequest(func(client *http.Client, url string, data []byte, apiToken string) (Response, error) {
		if i < maxRetries+1 {
			i++
			return Response{
				StatusCode: 400,
				Body:       []byte("{ err: 'failed to create thing'}"),
			}, nil
		}
		return Response{
			StatusCode: 200,
			Body:       []byte("Success"),
		}, nil
	})

	_, err := retry(nil, mockCall, "dont matter", "some/path", []byte("body"), "token", maxRetries, 1)
	assert.Check(t, err != nil)
	assert.ErrorContains(t, err, "400")
	assert.ErrorContains(t, err, "{ err: 'failed to create thing'}")
}

type testServerResponse struct {
	statusCode int
	body       string
}

type testQueryParams struct {
	key   string
	value string
}

func Test_GetObjectIdIfAlreadyExists_WorksCorrectlyForAddedQueryParameters(t *testing.T) {

	tests := []struct {
		name                          string
		apiKey                        string
		expectedQueryParamsPerApiCall [][]testQueryParams
		expectedApiCalls              int
		serverResponses               []testServerResponse
		expectError                   bool
	}{
		{
			name:                          "Sends no special query params for normal API",
			expectedQueryParamsPerApiCall: [][]testQueryParams{},
			expectedApiCalls:              1,
			serverResponses: []testServerResponse{
				{200, `{ "values": [ {"id": "1", "name": "name1"} ] }`},
			},
			apiKey:      "random-api", //not testing a real API, so this won't break if params are ever added to one
			expectError: false,
		},
		{
			name:                          "Returns error if HTTP error is encountered",
			expectedQueryParamsPerApiCall: [][]testQueryParams{},
			expectedApiCalls:              1,
			serverResponses: []testServerResponse{
				{400, `epic fail`},
			},
			apiKey:      "random-api", //not testing a real API, so this won't break if params are ever added to one
			expectError: true,
		},
		{
			name: "Returns error if HTTP error is encountered getting further paginated responses",
			expectedQueryParamsPerApiCall: [][]testQueryParams{
				{},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			expectedApiCalls: 5,
			serverResponses: []testServerResponse{
				{200, `{ "nextPageKey": "page42", "values": [ {"id": "1", "name": "name1"} ] }`},
				{400, `epic fail`}, // fail paginated request
				{400, `epic fail`}, // still fail after retry
				{400, `epic fail`}, // still fail after 2nd retry
				{400, `epic fail`}, // still fail after 3rd retry
			},
			apiKey:      "slo",
			expectError: true,
		},
		{
			name: "Retries on HTTP error on paginated request and returns eventual success",
			expectedQueryParamsPerApiCall: [][]testQueryParams{
				{},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			expectedApiCalls: 4,
			serverResponses: []testServerResponse{
				{200, `{ "nextPageKey": "page42", "values": [ {"id": "1", "name": "name1"} ] }`},
				{400, `epic fail`}, // fail paginated request
				{400, `epic fail`}, // still fail after retry
				{200, `{ "values": [ {"id": "1", "name": "name1"} ] }`},
			},
			apiKey:      "random-api", //not testing a real API, so this won't break if params are ever added to one
			expectError: false,
		},
		{
			name: "Sends correct param to get all SLOs",
			expectedQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"enabledSlos", "all"},
				},
			},
			expectedApiCalls: 1,
			serverResponses: []testServerResponse{
				{200, `{ "values": [ {"id": "1", "name": "name1"} ] }`},
			},
			apiKey:      "slo",
			expectError: false,
		},
		{
			name: "Sends correct parameters for paginated SLO responses",
			expectedQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"enabledSlos", "all"},
				},
				{
					{"nextPageKey", "page42"},
				},
				{
					{"nextPageKey", "page43"},
				},
			},
			expectedApiCalls: 3,
			serverResponses: []testServerResponse{
				{200, `{ "nextPageKey": "page42", "values": [ {"id": "1", "name": "name1"} ] }`},
				{200, `{ "nextPageKey": "page43", "values": [ {"id": "2", "name": "name2"} ] }`},
				{200, `{ "values": [ {"id": "3", "name": "name3"} ] }`},
			},
			apiKey:      "slo",
			expectError: false,
		},
		{
			name: "Sends correct param to get all anomaly detection metrics",
			expectedQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"includeEntityFilterMetricEvents", "true"},
				},
			},
			expectedApiCalls: 1,
			serverResponses: []testServerResponse{
				{200, `{ "values": [ {"id": "1", "name": "name1"} ] }`},
			},
			apiKey:      "anomaly-detection-metrics",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.expectedQueryParamsPerApiCall) > 0 {
					params := tt.expectedQueryParamsPerApiCall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.Assert(t, addedQueryParameter != nil)
						assert.Assert(t, len(addedQueryParameter) > 0)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.serverResponses[apiCalls]
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}

				apiCalls++
				assert.Check(t, apiCalls <= tt.expectedApiCalls, "expected at most %d API calls to happen, but encountered call %d", tt.expectedApiCalls, apiCalls)
			}))
			defer server.Close()
			testApi := api.NewStandardApi(tt.apiKey, "", false, "")
			_, err := getObjectIdIfAlreadyExists(server.Client(), testApi, server.URL, "", "")

			if tt.expectError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
			}

			assert.Equal(t, apiCalls, tt.expectedApiCalls, "expected exactly %d API calls to happen but %d calls where made", tt.expectedApiCalls, apiCalls)
		})

	}
}
