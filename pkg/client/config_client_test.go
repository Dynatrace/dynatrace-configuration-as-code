//go:build unit

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

package client

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testReportsApi = api.API{ID: "reports", URLPath: "/api/config/v1/reports"}
var testDashboardApi = api.API{ID: "dashboard", URLPath: "/api/config/v1/dashboards", NonUniqueName: true, DeprecatedBy: "dashboard-v2"}
var testMobileAppApi = api.API{ID: "application-mobile", URLPath: "/api/config/v1/applications/mobile"}
var testServiceDetectionApi = api.API{ID: "service-detection-full-web-request", URLPath: "/api/config/v1/service/detectionRules/FULL_WEB_REQUEST"}
var testSyntheticApi = api.API{ID: "synthetic-monitor", URLPath: "/api/environment/v1/synthetic/monitor"}

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

func Test_success(t *testing.T) {
	tests := []struct {
		name string
		resp rest.Response
		want bool
	}{
		{
			"200 is success",
			rest.Response{
				StatusCode: 200,
			},
			true,
		},
		{
			"201 is success",
			rest.Response{
				StatusCode: 201,
			},
			true,
		},
		{
			"401 is NOT success",
			rest.Response{
				StatusCode: 401,
			},
			false,
		},
		{
			"503 is NOT success",
			rest.Response{
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

func Test_isApplicationNotReadyYet(t *testing.T) {
	type args struct {
		resp   rest.Response
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
				rest.Response{
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
				rest.Response{
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
				rest.Response{
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
				rest.Response{
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

			got, err := getObjectIdIfAlreadyExists(server.Client(), testApi, server.URL, tt.givenObjectName, testRetrySettings)
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
			testApi := api.API{ID: tt.apiKey}
			s := rest.RetrySettings{
				Normal: rest.RetrySetting{
					WaitTime:   0,
					MaxRetries: 3,
				},
			}
			_, err := getObjectIdIfAlreadyExists(server.Client(), testApi, server.URL, "", s)

			if tt.expectError {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
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
	}{
		{
			name:                "Calls correct POST endpoint",
			objectName:          "Test object",
			apiKey:              "dashboard",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 200, body: `{ "id": "42", "name": "Test object" }`},
			want:                DynatraceEntity{Id: "42", Name: "Test object"},
			wantErr:             false,
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
		},
		{
			name:                "Returns err on server error",
			objectName:          "Test object",
			apiKey:              "auto-tag",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 400, body: `{}`},
			want:                DynatraceEntity{},
			wantErr:             true,
		},
		{
			name:                "Returns error if response can't be parsed",
			objectName:          "Test object",
			apiKey:              "auto-tag",
			expectedQueryParams: []testQueryParams{},
			serverResponse:      testServerResponse{statusCode: 200, body: `{ "not": "a value" }`},
			want:                DynatraceEntity{},
			wantErr:             true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.expectedQueryParams) > 0 {

					for _, param := range tt.expectedQueryParams {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.Assert(t, addedQueryParameter != nil)
						assert.Assert(t, len(addedQueryParameter) > 0)
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

			got, err := createDynatraceObject(server.Client(), server.URL, tt.objectName, testApi, []byte("{}"), testRetrySettings)
			if (err != nil) != tt.wantErr {
				t.Errorf("createDynatraceObject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.DeepEqual(t, got, tt.want)
		})
	}
}

func TestDeployConfigsTargetingClassicConfigNonUnique(t *testing.T) {
	theConfigName := "theConfigName"
	theCfgId := "monaco_cfg_id"
	theProject := "project"

	generatedUuid := idutils.GenerateUuidFromConfigId(theProject, theCfgId)

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

			got, err := upsertDynatraceEntityByNonUniqueNameAndId(server.Client(), server.URL, generatedUuid, theConfigName, testApi, []byte("{}"), testRetrySettings)
			assert.NilError(t, err)
			assert.Equal(t, got.Id, tt.expectedIdToBeUpserted)
		})
	}
}
