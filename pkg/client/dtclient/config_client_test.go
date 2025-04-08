//go:build unit

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

package dtclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

var mockAPI = api.API{ID: "mock-api", SingleConfiguration: true}
var mockAPISlo = api.API{ID: api.Slo, SingleConfiguration: true}
var mockAPINotSingle = api.API{ID: "mock-api", SingleConfiguration: false}

var testReportsApi = api.API{ID: "reports", URLPath: "/api/config/v1/reports"}
var testDashboardApi = api.API{ID: "dashboard", URLPath: "/api/config/v1/dashboards", NonUniqueName: true}
var testMobileAppApi = api.API{ID: "application-mobile", URLPath: "/api/config/v1/applications/mobile"}
var testServiceDetectionApi = api.API{ID: "service-detection-full-web-request", URLPath: "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST"}
var testSyntheticApi = api.API{ID: "synthetic-monitor", URLPath: "/api/environment/v1/synthetic/monitor"}

var testNetworkZoneApi = api.API{ID: "network-zone"}

var retrySettings = RetrySettings{
	Normal: RetrySetting{
		WaitTime:   0,
		MaxRetries: 3,
	},
}

func TestTranslateGenericValuesOnStandardResponse(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(t.Context(), response, "extensions")

	assert.NoError(t, err)
	assert.Len(t, values, 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "bar")
}

func TestTranslateGenericValuesOnIdMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	_, err := translateGenericValues(t.Context(), response, "extensions")

	assert.ErrorContains(t, err, "config of type extensions was invalid: No id")
}

func TestTranslateGenericValuesOnNameMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(t.Context(), response, "extensions")

	assert.NoError(t, err)
	assert.Len(t, values, 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "foo")
}

func TestTranslateGenericValuesForReportsEndpoint(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"
	entry["dashboardId"] = "dashboardId"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(t.Context(), response, "reports")

	assert.NoError(t, err)
	assert.Len(t, values, 1)

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

	testWebApi := api.API{ID: "application-web", URLPath: "/api/config/v1/applications/web"}
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

func Test_isApplicationNotReadyYet(t *testing.T) {
	type args struct {
		resp   coreapi.APIError
		theApi api.API
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Server Error on synthetic counted as app not ready (issue in error reporting for unknown App IDs in some Dynatrace versions)",
			args{
				coreapi.APIError{
					StatusCode: 500,
					Body:       nil,
				},
				testSyntheticApi,
			},
			true,
		},
		{
			"Server Error on application API counts as not ready (can happen on update)",
			args{
				coreapi.APIError{
					StatusCode: 503,
					Body:       nil,
				},
				testMobileAppApi,
			},
			true,
		},
		{
			"Server Error on unexpected API not counted as App not ready",
			args{
				coreapi.APIError{
					StatusCode: 503,
					Body:       nil,
				},
				testDashboardApi,
			},
			false,
		},
		{
			"User error response of 'Unknown Application' counted as not ready (can happen if App was just created)",
			args{
				coreapi.APIError{
					StatusCode: 400,
					Body:       []byte("Unknown application(s) APP-422142"),
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

func Test_isNetworkZoneFeatureNotEnabledYet(t *testing.T) {
	type args struct {
		resp   coreapi.APIError
		theApi api.API
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"HTTP 400: Network zone feature disabled",
			args{
				coreapi.APIError{
					StatusCode: 400,
					Body:       []byte("Not allowed because network zones are disabled"),
				},
				testNetworkZoneApi,
			},
			true,
		},
		{
			"HTTP 400: Another Error",
			args{
				coreapi.APIError{
					StatusCode: 400,
					Body:       []byte("Something bad"),
				},
				testNetworkZoneApi,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkZoneFeatureNotEnabledYet(tt.args.resp, tt.args.theApi); got != tt.want {
				t.Errorf("isNetworkZoneFeatureNotEnabledYet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getObjectIdIfAlreadyExists(t *testing.T) {

	testApi := api.API{ID: "test", URLPath: "/test/api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}

	tests := []struct {
		name                    string
		givenObjectName         string
		givenApiResponse        string
		givenApiResponseIsError bool
		wantFoundId             string
		wantErr                 bool
	}{
		{
			name:             "finds object id as expected",
			givenObjectName:  "TEST NAME",
			givenApiResponse: `{ "values": [ { "id": "42", "name": "TEST NAME" } ] }`,
			wantFoundId:      "42",
			wantErr:          false,
		},
		{
			name:             "returns first match if more than one object of given name exist",
			givenObjectName:  "TEST NAME",
			givenApiResponse: `{ "values": [ { "id": "41", "name": "TEST NAME" }, { "id": "42", "name": "TEST NAME" } ] }`,
			wantFoundId:      "41",
			wantErr:          false,
		},
		{
			name:             "returns empty string without error if no match found",
			givenObjectName:  "TEST NAME",
			givenApiResponse: `{ "values": [ { "id": "42", "name": "some other thing" } ] }`,
			wantFoundId:      "",
			wantErr:          false,
		},
		{
			name:             "returns object id as expected if string escaping is needed to match",
			givenObjectName:  `TEST \"NAME\"`,
			givenApiResponse: `{ "values": [ { "id": "42", "name": "TEST \"NAME\"" } ] }`, // note after API GET and unmarshalling this will be 'TEST "NAME"' and not match directly
			wantFoundId:      "42",
			wantErr:          false,
		},
		{
			name:                    "returns error if API call fails",
			givenObjectName:         "TEST NAME",
			givenApiResponseIsError: true,
			wantFoundId:             "",
			wantErr:                 true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if tt.givenApiResponseIsError {
					rw.WriteHeader(400)
					return
				}
				_, _ = rw.Write([]byte(tt.givenApiResponse))
			}))
			defer server.Close()

			dtclient, _ := NewClassicConfigClientForTesting(server.URL, server.Client(), WithRetrySettingsForClassic(retrySettings))
			_, got, err := dtclient.ExistsWithName(t.Context(), testApi, tt.givenObjectName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getObjectIdIfAlreadyExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantFoundId {
				t.Errorf("getObjectIdIfAlreadyExists() got = %v, want %v", got, tt.wantFoundId)
			}
		})
	}
}

func TestUpsertByName(t *testing.T) {
	tests := []struct {
		name             string
		testApi          api.API
		givenApiResponse string
		expectedAPIHits  int
	}{
		{
			name:             "cache is used for fetching existing values",
			testApi:          api.API{ID: "test", URLPath: "/test/api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse},
			givenApiResponse: `{ "values": [ { "id": "42", "name": "MY CONFIG" }, {"id": "43", "name": "MY CONFIG 2" } ] }`,
			expectedAPIHits:  3, // one for getting existing values, one for updating MY CONFIG and one for updating MY CONFIG 2
		},
		{
			name:             "cache is not used for fetching existing values when dealing with non unique name configs",
			testApi:          api.API{ID: "test", URLPath: "/test/api", NonUniqueName: true, PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse},
			givenApiResponse: `{ "values": [ { "id": "42", "name": "MY CONFIG" }, {"id": "43", "name": "MY CONFIG 2" } ] }`,
			expectedAPIHits:  4, // one for getting existing values, one for updating MY CONFIG another one for getting existing values and one for updating MY CONFIG 2
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiHits := 0
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				apiHits++
				rw.Write([]byte(tt.givenApiResponse))
				rw.WriteHeader(http.StatusOK)
				fmt.Println(req.URL)
			}))
			defer server.Close()

			dtClient, _ := NewClassicConfigClientForTesting(server.URL, server.Client())
			dtClient.UpsertByName(t.Context(), tt.testApi, "MY CONFIG", nil)
			dtClient.UpsertByName(t.Context(), tt.testApi, "MY CONFIG 2", nil)
			assert.Equal(t, apiHits, tt.expectedAPIHits)
		})
	}
}

func TestUpsertConfig_CheckEqualityFunctionIsUsed(t *testing.T) {
	tests := []struct {
		name                     string
		testApi                  api.API
		fetchExistingAPIResponse string
		createAPIResponse        string
		updateAPIResponse        string
		expectedDynatraceObject  DynatraceEntity
		expectedAPIHits          int
	}{
		{
			name:                     "existing object found with custom function - update",
			testApi:                  api.API{ID: "test", URLPath: "/test/api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse, CheckEqualFunc: func(_ map[string]any, _ map[string]any) bool { return true }},
			fetchExistingAPIResponse: `{ "values": [ { "id": "42", "name": "MY CONFIG" } ] }`,
			updateAPIResponse:        `{ "id": "42", "name": "MY NEW CONFIG" }`,
			expectedAPIHits:          2,
			expectedDynatraceObject:  DynatraceEntity{Id: "42", Name: "MY CONFIG", Description: "Updated existing object"},
		},
		{
			name:                     "no existing object found with custom function - create",
			testApi:                  api.API{ID: "test", URLPath: "/test/api", PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse, CheckEqualFunc: func(_ map[string]any, _ map[string]any) bool { return false }},
			fetchExistingAPIResponse: `{ "values": [ { "id": "42", "name": "MY CONFIG" } ] }`,
			createAPIResponse:        `{ "id": "44", "name": "MY NEW CONFIG" }`,
			expectedAPIHits:          2,
			expectedDynatraceObject:  DynatraceEntity{Id: "44", Name: "MY NEW CONFIG"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiHits := 0
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				apiHits++
				if req.Method == http.MethodGet {
					rw.Write([]byte(tt.fetchExistingAPIResponse))
					rw.WriteHeader(http.StatusOK)
				}
				if req.Method == http.MethodPost {
					rw.Write([]byte(tt.createAPIResponse))
					rw.WriteHeader(http.StatusOK)
				}
				if req.Method == http.MethodPut {
					rw.Write([]byte(tt.updateAPIResponse))
					rw.WriteHeader(http.StatusOK)
				}

			}))
			defer server.Close()

			dtClient, _ := NewClassicConfigClientForTesting(server.URL, server.Client())
			dtObj, err := dtClient.UpsertByName(t.Context(), tt.testApi, "MY CONFIG", []byte(`{}`))
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAPIHits, 2)
			assert.Equal(t, tt.expectedDynatraceObject, dtObj)
		})
	}
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
			expectedApiCalls:              4,
			serverResponses: []testServerResponse{
				{400, `epic fail`},
				{400, `epic fail`},
				{400, `epic fail`},
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
				{http.StatusGone, `epic fail`}, // fail paginated request
				{http.StatusGone, `epic fail`}, // still fail after retry
				{http.StatusGone, `epic fail`}, // still fail after 2nd retry
				{http.StatusGone, `epic fail`}, // still fail after 3rd retry
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
		fmt.Println(tt.name)
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.expectedQueryParamsPerApiCall) > 0 {
					params := tt.expectedQueryParamsPerApiCall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.NotNil(t, addedQueryParameter)
						assert.Greater(t, len(addedQueryParameter), 0)
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
				assert.LessOrEqual(t, apiCalls, tt.expectedApiCalls, "expected at most %d API calls to happen, but encountered call %d", tt.expectedApiCalls, apiCalls)
			}))
			defer server.Close()
			testApi := api.API{ID: tt.apiKey}
			dtclient, _ := NewClassicConfigClientForTesting(server.URL, server.Client(), WithRetrySettingsForClassic(retrySettings))

			_, _, err := dtclient.ExistsWithName(t.Context(), testApi, "")
			if tt.expectError {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, apiCalls, tt.expectedApiCalls, "expected exactly %d API calls to happen but %d calls where made", tt.expectedApiCalls, apiCalls)
		})

	}
}

func Test_createDynatraceObject(t *testing.T) {
	tests := []struct {
		name                string
		objectName          string
		apiKey              string
		expectedQueryParams []testQueryParams
		serverResponse      testServerResponse
		want                DynatraceEntity
		wantErr             bool
		payload             []byte
	}{
		{
			name:                "Calls correct POST endpoint",
			objectName:          "Test object",
			apiKey:              "dashboard",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 200, body: `{ "id": "42", "name": "Test object" }`},
			want:                DynatraceEntity{Id: "42", Name: "Test object"},
			wantErr:             false,
			payload:             []byte("{}"),
		},
		{
			name:       "Sends expected query parameters when creating app-detection-rule",
			objectName: "Test object",
			apiKey:     "app-detection-rule",
			expectedQueryParams: []testQueryParams{
				{
					key:   "position",
					value: "PREPEND",
				},
			},
			serverResponse: testServerResponse{statusCode: 200, body: `{ "id": "42", "name": "Test object" }`},
			want:           DynatraceEntity{Id: "42", Name: "Test object"},
			wantErr:        false,
			payload:        []byte("{}"),
		},
		{
			name:                "Sends slo if payload is valid",
			objectName:          "Test object",
			apiKey:              api.Slo,
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 200, body: `{ "id": "42", "name": "Test object" }`},
			want:                DynatraceEntity{Id: "42", Name: "Test object"},
			wantErr:             false,
			payload:             []byte(`{"evaluationType": "AGGREGATE"}`),
		},
		{
			name:                "Returns err on server error",
			objectName:          "Test object",
			apiKey:              "auto-tag",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 400, body: `{}`},
			want:                DynatraceEntity{},
			wantErr:             true,
			payload:             []byte("{}"),
		},
		{
			name:                "Returns error if response can't be parsed",
			objectName:          "Test object",
			apiKey:              "auto-tag",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 200, body: `{ "not": "a value" }`},
			want:                DynatraceEntity{},
			wantErr:             true,
			payload:             []byte("{}"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.expectedQueryParams) > 0 {

					for _, param := range tt.expectedQueryParams {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.NotNil(t, addedQueryParameter)
						assert.Greater(t, len(addedQueryParameter), 0)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.serverResponse
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}
			}))
			defer server.Close()
			testApi := api.API{ID: tt.apiKey}

			dtclient, _ := NewClassicConfigClientForTesting(server.URL, server.Client(), WithRetrySettingsForClassic(testRetrySettings))
			got, err := dtclient.createDynatraceObject(t.Context(), tt.objectName, testApi, tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDynatraceObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestDeployConfigsTargetingClassicConfigNonUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theCfgId := "monaco_cfg_id"
	theProject := "project"

	generatedUuid := idutils.GenerateUUIDFromConfigId(theProject, theCfgId)

	tests := []struct {
		name                   string
		existingValues         string
		expectedIdToBeUpserted string
	}{
		{
			name:                   "upserts new config",
			existingValues:         `{ "values": [] }`,
			expectedIdToBeUpserted: generatedUuid,
		},
		{
			name:                   "upserts new config with existing duplicate names",
			existingValues:         fmt.Sprintf(`{"values": [{ "id": "42", "name": "%s" }, { "id": "43", "name": "%s" }, { "id": "44", "name": "%s" }, { "id": "45", "name": "%s" }]}`, theConfigName, theConfigName, theConfigName, theConfigName),
			expectedIdToBeUpserted: generatedUuid,
		},
		{
			name:                   "updates config with exact match",
			existingValues:         fmt.Sprintf(`{"values": [{ "id": "42", "name": "%s" }, { "id": "%s", "name": "%s" }]}`, theConfigName, generatedUuid, theConfigName),
			expectedIdToBeUpserted: generatedUuid,
		},
		{
			name:                   "updates single known config with name is currently unique",
			existingValues:         fmt.Sprintf(`{"values": [{ "id": "42", "name": "%s" }, { "id": "43", "name": "some_other_config" }]}`, theConfigName),
			expectedIdToBeUpserted: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				_, _ = rw.Write([]byte(tt.existingValues))
			}))
			defer server.Close()

			testApi := api.API{ID: "some-api", NonUniqueName: true, PropertyNameOfGetAllResponse: api.StandardApiPropertyNameOfGetAllResponse}

			dtclient, _ := NewClassicConfigClientForTesting(server.URL, server.Client(), WithRetrySettingsForClassic(testRetrySettings))
			got, err := dtclient.UpsertByNonUniqueNameAndId(t.Context(), testApi, generatedUuid, theConfigName, []byte("{}"), false)
			assert.NoError(t, err)
			assert.Equal(t, got.Id, tt.expectedIdToBeUpserted)
		})
	}
}

func TestReadByIdReturnsAnErrorUponEncounteringAnError(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	_, err = client.Get(t.Context(), mockAPI, "test")
	assert.ErrorContains(t, err, "failed with status code")
}

func TestReadByIdEscapesTheId(t *testing.T) {
	unescapedID := "ruxit.perfmon.dotnetV4:%TimeInGC:time_in_gc_alert_high_generic"

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	_, err = client.Get(t.Context(), mockAPINotSingle, unescapedID)
	assert.NoError(t, err)
}

func TestReadByIdReturnsTheResponseGivenNoError(t *testing.T) {
	body := []byte{1, 3, 3, 7}

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write(body)
	}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	resp, err := client.Get(t.Context(), mockAPI, "test")
	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, body, resp)
}

func TestSloV2ToSloV1(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	_, err = client.UpsertByName(t.Context(), mockAPISlo, "test", []byte("{}"))
	assert.ErrorContains(t, err, "tried to deploy an slo-v2 configuration to slo-v1")
}

func TestDocumentV2ToDocumentV1_Fails(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	_, err = client.UpsertByName(t.Context(), testDashboardApi, "test", []byte(`{
		"tiles": {
			"1": {
			  "content": "my content",
			  "title": "",
			  "type": "markdown"
			}
		}
	}`))
	assert.ErrorContains(t, err, "tried to deploy")
}

func TestValidateDashboardPayload(t *testing.T) {
	v2Payload := []byte(`{
		"tiles": {
			"1": {
			  "content": "my content",
			  "title": "",
			  "type": "markdown"
			}
		}
	}`)
	v1Payload := []byte(`{
		"tiles": [
			{
			  "name": "Host health",
			  "tileType": "HOSTS",
			  "configured": true,
			  "bounds": {
				"top": 0,
				"left": 38,
				"width": 304,
				"height": 304
			  },
			  "tileFilter": {
				"timeframe": null,
				"managementZone": null
			  },
			  "filterConfig": null,
			  "chartVisible": true
			}
		]
	}`)
	testcases := []struct {
		payload   []byte
		isV1      bool
		expectErr bool
		name      string
	}{
		{
			name:      "Should not return any error if payload is V1 and expected is V1",
			payload:   v1Payload,
			isV1:      true,
			expectErr: false,
		},
		{
			name:      "Should not return any error if payload is V2 and expected is V2",
			payload:   v2Payload,
			isV1:      false,
			expectErr: false,
		},
		{
			name:      "Should return an error if payload is V1 and expected is V2",
			payload:   v1Payload,
			isV1:      false,
			expectErr: true,
		},
		{
			name:      "Should return an error if payload is V2 and expected is V1",
			payload:   v2Payload,
			isV1:      true,
			expectErr: true,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDashboardPayload(tt.payload, tt.isV1)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCallWithRetryOnKnowTimingIssue_IgnoreRetryOn(t *testing.T) {
	testServer := httptest.NewTLSServer(http.NewServeMux())
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	i := 0
	mockCall := SendRequestWithBody(func(ctx context.Context, url string, data io.Reader, options corerest.RequestOptions) (*http.Response, error) {
		i++
		return nil, coreapi.APIError{StatusCode: http.StatusForbidden, Body: []byte("Metric selector invalid")}
	})
	_, err = client.callWithRetryOnKnowTimingIssue(t.Context(), mockCall, "some/path", []byte("{}"), api.API{}, corerest.RequestOptions{})
	assert.Error(t, err)
	assert.Equal(t, 1, i)
}

func TestConfigClient_ClearCache(t *testing.T) {
	body := Value{
		Id:           "c-id",
		Name:         "config-name",
		Owner:        nil,
		Type:         nil,
		CustomFields: nil,
	}
	listCalledCount := 0
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		listCalledCount++
		data, err := json.Marshal(body)
		require.NoError(t, err)

		_, err = res.Write(data)
		require.NoError(t, err)
	}))
	defer testServer.Close()

	client, err := NewClassicConfigClientForTesting(testServer.URL, testServer.Client())
	require.NoError(t, err)

	// trigger an API call and save to cache
	_, err = client.List(t.Context(), mockAPISlo)
	require.NoError(t, err)
	require.Equal(t, listCalledCount, 1)

	// check if cache was used and no API call was triggered
	_, err = client.List(t.Context(), mockAPISlo)
	require.NoError(t, err)
	require.Equal(t, listCalledCount, 1)

	// clear cache and check if API call is triggered
	client.ClearCache()
	_, err = client.List(t.Context(), mockAPISlo)
	require.NoError(t, err)
	require.Equal(t, listCalledCount, 2)
}
