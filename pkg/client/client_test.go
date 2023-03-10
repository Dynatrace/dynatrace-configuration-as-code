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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"gotest.tools/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockApi = api.NewApi("mock-api", "/mock-api", "", true, true, "", false)
var mockApiNotSingle = api.NewApi("mock-api", "/mock-api", "", false, true, "", false)

func TestNewClientNoUrl(t *testing.T) {
	_, err := NewDynatraceClient(nil, "")
	assert.ErrorContains(t, err, "empty url")
}

func TestNewClientInvalidURL(t *testing.T) {
	_, err := NewDynatraceClient(nil, "INVALID_URL")
	assert.ErrorContains(t, err, "not valid")
}

func TestUrlSuffixGetsTrimmed(t *testing.T) {
	client, err := NewDynatraceClient(nil, "https://my-environment.live.dynatrace.com/")
	assert.NilError(t, err)
	assert.Equal(t, client.environmentUrl, "https://my-environment.live.dynatrace.com")
}

func TestNewDynatraceClientWithHTTP(t *testing.T) {
	client, err := NewDynatraceClient(nil, "http://my-environment.live.dynatrace.com")
	assert.NilError(t, err)
	assert.Equal(t, client.environmentUrl, "http://my-environment.live.dynatrace.com")
}

func TestNewDynatraceClientWithoutScheme(t *testing.T) {
	_, err := NewDynatraceClient(nil, "my-environment.live.dynatrace.com")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewDynatraceClientWithIPv4(t *testing.T) {
	client, err := NewDynatraceClient(nil, "https://127.0.0.1")
	assert.NilError(t, err)
	assert.Equal(t, client.environmentUrl, "https://127.0.0.1")
}

func TestNewDynatraceClientWithIPv6(t *testing.T) {
	client, err := NewDynatraceClient(nil, "https://[0000:0000:0000:0000:0000:0000:0000:0001]")
	assert.NilError(t, err)
	assert.Equal(t, client.environmentUrl, "https://[0000:0000:0000:0000:0000:0000:0000:0001]")
}

func TestNewClientNoValidUrlLocalPath(t *testing.T) {
	_, err := NewDynatraceClient(nil, "/my-environment/live/dynatrace.com/")
	assert.ErrorContains(t, err, "no host specified")
}

func TestNewClientNoValidUrlTypo(t *testing.T) {
	_, err := NewDynatraceClient(nil, "https//my-environment.live.dynatrace.com/")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewClientNoValidUrlNoHttps(t *testing.T) {
	_, err := NewDynatraceClient(nil, "http//my-environment.live.dynatrace.com/")
	assert.ErrorContains(t, err, "not valid")
}

func TestNewClient(t *testing.T) {
	httpClient := &http.Client{}
	c, err := NewDynatraceClient(httpClient, "https://my-environment.live.dynatrace.com/",
		WithServerVersion(version.Version{Major: 1, Minor: 2, Patch: 3}),
		WithRetrySettings(rest.DefaultRetrySettings))
	assert.Equal(t, version.Version{Major: 1, Minor: 2, Patch: 3}, c.serverVersion)
	assert.Equal(t, rest.DefaultRetrySettings, c.retrySettings)
	assert.Equal(t, httpClient, c.client)
	assert.Equal(t, "https://my-environment.live.dynatrace.com", c.environmentUrl)
	assert.NilError(t, err, "not valid")
}

func TestReadByIdReturnsAnErrorUponEncounteringAnError(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer func() { testServer.Close() }()
	client, _ := NewDynatraceClient(testServer.Client(), testServer.URL)

	_, err := client.ReadConfigById(mockApi, "test")
	assert.ErrorContains(t, err, "Response was")
}

func TestReadByIdEscapesTheId(t *testing.T) {
	unescapedId := "ruxit.perfmon.dotnetV4:%TimeInGC:time_in_gc_alert_high_generic"

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
	defer func() { testServer.Close() }()
	client, _ := NewDynatraceClient(testServer.Client(), testServer.URL)

	_, err := client.ReadConfigById(mockApiNotSingle, unescapedId)
	assert.NilError(t, err)
}

func TestReadByIdReturnsTheResponseGivenNoError(t *testing.T) {
	body := []byte{1, 3, 3, 7}

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write(body)
	}))
	defer func() { testServer.Close() }()

	client, _ := NewDynatraceClient(testServer.Client(), testServer.URL)

	resp, err := client.ReadConfigById(mockApi, "test")
	assert.NilError(t, err, "there should not be an error")
	assert.DeepEqual(t, body, resp)
}

func TestListKnownSettings(t *testing.T) {

	tests := []struct {
		name                      string
		givenSchemaId             string
		givenListSettingsOpts     ListSettingsOptions
		givenServerResponses      []testServerResponse
		want                      []DownloadSettingsObject
		wantQueryParamsPerApiCall [][]testQueryParams
		wantNumberOfApiCalls      int
		wantError                 bool
	}{
		{
			name:          "Lists Settings objects as expected",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:                  "Lists Settings objects without value field as expected",
			givenSchemaId:         "builtin:something",
			givenListSettingsOpts: ListSettingsOptions{DiscardValue: true},
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", reducedListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:          "Lists Settings objects with filter as expected",
			givenSchemaId: "builtin:something",
			givenListSettingsOpts: ListSettingsOptions{Filter: func(o DownloadSettingsObject) bool {
				return o.ExternalId == "RG9jdG9yIFdobwo="
			}},
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4641", "externalId": "RG9jdG9yIabcdef="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:          "Handles Pagination when listing settings objects",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
				{
					ExternalId: "VGhlIE1hc3Rlcgo=",
					ObjectId:   "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
				},
			},

			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 2,
			wantError:            false,
		},
		{
			name:          "Returns empty if list if no items exist",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ ] }`},
			},
			want: []DownloadSettingsObject{},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered - 400",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            true,
		},
		{
			name:          "Returns error if HTTP error is encountered - 403",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{403, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            true,
		},
		{
			name:          "Retries on HTTP error on paginated request and returns eventual success",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail`},
				{200, `{ "items": [ {"objectId": "b1d4c623-25e0-4b54-9eb5-6734f1a72041", "externalId": "VGhlIE1hc3Rlcgo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
				{
					ExternalId: "VGhlIE1hc3Rlcgo=",
					ObjectId:   "b1d4c623-25e0-4b54-9eb5-6734f1a72041",
				},
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
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
			wantNumberOfApiCalls: 4,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered getting further paginated responses",
			givenSchemaId: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail 1`},
				{400, `retry fail 2`},
				{400, `retry fail 3`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
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
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 5,
			wantError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerApiCall) > 0 {
					params := tt.wantQueryParamsPerApiCall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.Assert(t, addedQueryParameter != nil)
						assert.Assert(t, len(addedQueryParameter) > 0)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.givenServerResponses[apiCalls]
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}

				apiCalls++
				assert.Check(t, apiCalls <= tt.wantNumberOfApiCalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfApiCalls, apiCalls)
			}))
			defer server.Close()

			client, err := NewDynatraceClient(server.Client(), server.URL, WithRetrySettings(testRetrySettings))
			assert.NilError(t, err)

			res, err1 := client.ListSettings(tt.givenSchemaId, tt.givenListSettingsOpts)

			if tt.wantError {
				assert.Assert(t, err1.WrappedError != nil)
			} else {
				assert.NilError(t, err1.WrappedError)
			}

			assert.DeepEqual(t, res, tt.want)

			assert.Equal(t, apiCalls, tt.wantNumberOfApiCalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfApiCalls, apiCalls)
		})
	}
}

func TestGetSettingById(t *testing.T) {
	type fields struct {
		environmentUrl string
		client         *http.Client
		retrySettings  rest.RetrySettings
	}
	type args struct {
		objectID string
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		givenTestServerResp *testServerResponse
		wantURLPath         string
		wantResult          *DownloadSettingsObject
		wantErr             bool
	}{
		{
			name: "Get Setting by ID - malformed environment URL",
			fields: fields{
				environmentUrl: " https://leading-space.com",
			},
			args:        args{},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult:  nil,
			wantErr:     true,
		},
		{
			name:   "Get Setting by ID - server response != 2xx",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 500,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult:  nil,
			wantErr:     true,
		},
		{
			name:   "Get Setting by ID - invalid server response",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 200,
				body:       `{bs}`,
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult:  nil,
			wantErr:     true,
		},
		{
			name:   "Get Setting by ID",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 200,
				body:       `{"objectId":"12345","externalId":"54321", "schemaVersion":"1.0","schemaId":"builtin:bla","scope":"tenant"}`,
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantResult: &DownloadSettingsObject{
				ExternalId:    "54321",
				SchemaVersion: "1.0",
				SchemaId:      "builtin:bla",
				ObjectId:      "12345",
				Scope:         "tenant",
				Value:         nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, tt.wantURLPath, req.URL.Path)
				if resp := tt.givenTestServerResp; resp != nil {
					if resp.statusCode != 200 {
						http.Error(rw, resp.body, resp.statusCode)
					} else {
						_, _ = rw.Write([]byte(resp.body))
					}
				}

			}))
			defer server.Close()

			var envURL string
			if tt.fields.environmentUrl != "" {
				envURL = tt.fields.environmentUrl
			} else {
				envURL = server.URL
			}

			d := &DynatraceClient{
				environmentUrl: envURL,
				client:         server.Client(),
				retrySettings:  tt.fields.retrySettings,
			}

			settingsObj, err := d.GetSettingById(tt.args.objectID)
			if tt.wantErr {
				assert.Assert(t, err != nil)
			} else {
				assert.NilError(t, err)
			}
			assert.DeepEqual(t, tt.wantResult, settingsObj)

		})
	}

}

func TestDeleteSettings(t *testing.T) {
	type fields struct {
		environmentUrl string
		client         *http.Client
		retrySettings  rest.RetrySettings
	}
	type args struct {
		objectID string
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		givenTestServerResp *testServerResponse
		wantURLPath         string
		wantErr             bool
	}{
		{
			name: "Delete Settings - malformed environment URL",
			fields: fields{
				environmentUrl: " https://leading-space.com",
			},
			args:        args{},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     true,
		},
		{
			name:   "Delete Settings - server response != 2xx",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 500,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     true,
		},
		{
			name:   "Delete Settings - server response 404 does not result in an err",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			givenTestServerResp: &testServerResponse{
				statusCode: 404,
				body:       "{}",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     false,
		},
		{
			name:   "Delete Settings - object ID is passed",
			fields: fields{},
			args: args{
				objectID: "12345",
			},
			wantURLPath: "/api/v2/settings/objects/12345",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				assert.Equal(t, tt.wantURLPath, req.URL.Path)
				if resp := tt.givenTestServerResp; resp != nil {
					if resp.statusCode != 200 {
						http.Error(rw, resp.body, resp.statusCode)
					} else {
						_, _ = rw.Write([]byte(resp.body))
					}
				}

			}))
			defer server.Close()

			var envURL string
			if tt.fields.environmentUrl != "" {
				envURL = tt.fields.environmentUrl
			} else {
				envURL = server.URL
			}

			d := &DynatraceClient{
				environmentUrl: envURL,
				client:         server.Client(),
				retrySettings:  tt.fields.retrySettings,
			}
			if err := d.DeleteSettings(tt.args.objectID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertSettingsRetries(t *testing.T) {
	numApiCalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			rw.WriteHeader(200)
			rw.Write([]byte("{}"))
			return
		}

		numApiCalls++
		if numApiCalls < 3 {
			rw.WriteHeader(409)
			return
		}
		rw.WriteHeader(200)
		rw.Write([]byte(`[{"objectId": "abcdefg"}]`))
	}))
	defer server.Close()

	client, err := NewDynatraceClient(server.Client(), server.URL, WithRetrySettings(testRetrySettings))
	assert.NilError(t, err)

	_, err = client.UpsertSettings(SettingsObject{
		Id:       "42",
		SchemaId: "some:schema",
		Content:  []byte("{}"),
	})

	assert.NilError(t, err)
	assert.Equal(t, numApiCalls, 3)
}

func TestListEntities(t *testing.T) {

	testType := "SOMETHING"

	tests := []struct {
		name                      string
		givenEntitiesType         EntitiesType
		givenServerResponses      []testServerResponse
		want                      []string
		wantQueryParamsPerApiCall [][]testQueryParams
		wantNumberOfApiCalls      int
		wantError                 bool
	}{
		{
			name:              "Lists Entities objects as expected",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ] }`, testType, testType)},
			},
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:              "Handles Pagination when listing entities objects",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-C329D7411A28B791", "type": "%s"} ] }`, testType, testType)},
			},
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
			},

			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 2,
			wantError:            false,
		},
		{
			name:              "Returns empty if list if no entities exist",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, `{ "entities": [ ] }`},
			},
			want: []string{},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            false,
		},
		{
			name:              "Returns error if HTTP error is encountered",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfApiCalls: 1,
			wantError:            true,
		},
		{
			name:              "Retries on HTTP error on paginated request and returns eventual success",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{400, `get next page fail`},
				{400, `retry fail`},
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-C329D7411A28B791", "type": "%s"} ] }`, testType, testType)},
			},
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
			},
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
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
			wantNumberOfApiCalls: 4,
			wantError:            false,
		},
		{
			name:              "Returns error if HTTP error is encountered getting further paginated responses",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{400, `get next page fail`},
				{400, `retry fail 1`},
				{400, `retry fail 2`},
				{400, `retry fail 3`},
			},
			want: nil,
			wantQueryParamsPerApiCall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
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
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfApiCalls: 5,
			wantError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerApiCall) > 0 {
					params := tt.wantQueryParamsPerApiCall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.Assert(t, addedQueryParameter != nil)
						assert.Assert(t, len(addedQueryParameter) > 0)
						assert.Equal(t, addedQueryParameter[0], param.value)
					}
				} else {
					assert.Equal(t, "", req.URL.RawQuery, "expected no query params - but '%s' was sent", req.URL.RawQuery)
				}

				resp := tt.givenServerResponses[apiCalls]
				if resp.statusCode != 200 {
					http.Error(rw, resp.body, resp.statusCode)
				} else {
					_, _ = rw.Write([]byte(resp.body))
				}

				apiCalls++
				assert.Check(t, apiCalls <= tt.wantNumberOfApiCalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfApiCalls, apiCalls)
			}))
			defer server.Close()

			client, err := NewDynatraceClient(server.Client(), server.URL, WithRetrySettings(testRetrySettings))
			assert.NilError(t, err)

			res, err1 := client.ListEntities(tt.givenEntitiesType)

			if tt.wantError {
				assert.Assert(t, err1.WrappedError != nil)
			} else {
				assert.NilError(t, err1.WrappedError)
			}

			assert.DeepEqual(t, res, tt.want)

			assert.Equal(t, apiCalls, tt.wantNumberOfApiCalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfApiCalls, apiCalls)
		})
	}
}

func TestCreateDynatraceClientWithAutoServerVersion(t *testing.T) {
	t.Run("Server version is correctly set to determined value", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Write([]byte(`{"version" : "1.262.0.20230214-193525"}`))
		}))

		dcl, err := NewDynatraceClient(server.Client(), server.URL, WithAutoServerVersion())
		server.Close()
		assert.NilError(t, err)
		assert.Equal(t, version.Version{Major: 1, Minor: 262}, dcl.serverVersion)
	})

	t.Run("Server version is correctly set to unknown", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Write([]byte(`{}`))
		}))

		dcl, err := NewDynatraceClient(server.Client(), server.URL, WithAutoServerVersion())
		server.Close()
		assert.NilError(t, err)
		assert.Equal(t, version.UnknownVersion, dcl.serverVersion)
	})
}
