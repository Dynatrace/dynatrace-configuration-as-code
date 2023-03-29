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
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

var mockAPI = api.API{ID: "mock-api", SingleConfiguration: true}
var mockAPINotSingle = api.API{ID: "mock-api", SingleConfiguration: false}

func TestNewClassicClient(t *testing.T) {
	t.Run("Create new classic client", func(t *testing.T) {
		dtURL := "https://some.url"

		ver := version.Version{Major: 1, Minor: 2, Patch: 3}

		c, err := NewClassicClient(dtURL, "",
			WithServerVersion(ver),
			WithRetrySettings(rest.DefaultRetrySettings))

		assert.NoError(t, err)

		assert.NotEmpty(t, c.serverVersion)
		assert.NotEmpty(t, c.environmentURL)
		assert.NotEmpty(t, c.environmentURL)
		assert.NotEmpty(t, c.client)
		assert.NotEmpty(t, c.clientClassic)
		assert.NotEmpty(t, c.retrySettings)
		assert.Equal(t, settingsSchemaAPIPathClassic, c.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathClassic, c.settingsObjectAPIPath)

		assert.Equal(t, dtURL, c.environmentURL)
		assert.Equal(t, c.environmentURL, c.environmentURLClassic, "'environmentURLClassic' should be same as 'environmentURL'")
		assert.Equal(t, c.client, c.clientClassic, "'classicClassic' should be same as 'client'")
		assert.Equal(t, ver, c.serverVersion, "'serverVersion' should be modified with 'WithServerVersion'")
		assert.Equal(t, rest.DefaultRetrySettings, c.retrySettings, "'retrySettings' should be modified with 'WithRetrySettings' modifier")
	})
	t.Run("URL is empty - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("", "")
		assert.ErrorContains(t, err, "empty url")
	})

	t.Run("invalid URL - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("INVALID_URL", "")
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL suffix is trimmed", func(t *testing.T) {
		client, err := NewClassicClient("https://my-environment.live.dynatrace.com/", "")
		assert.NoError(t, err)
		assert.Equal(t, client.environmentURL, "https://my-environment.live.dynatrace.com")
	})

	t.Run("URL with leading space - should return an error", func(t *testing.T) {
		_, err := NewClassicClient(" https://my-environment.live.dynatrace.com/", "")
		assert.Error(t, err)
	})

	t.Run("URL starts with http", func(t *testing.T) {
		client, err := NewClassicClient("http://my-environment.live.dynatrace.com", "")
		assert.NoError(t, err)
		assert.Equal(t, client.environmentURL, "http://my-environment.live.dynatrace.com")
	})

	t.Run("URL is without scheme - should throw an error", func(t *testing.T) {
		_, err := NewClassicClient("my-environment.live.dynatrace.com", "")
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL is IPv4", func(t *testing.T) {
		client, err := NewClassicClient("https://127.0.0.1", "")
		assert.NoError(t, err)
		assert.Equal(t, client.environmentURL, "https://127.0.0.1")
	})

	t.Run("URL is IPv6", func(t *testing.T) {
		client, err := NewClassicClient("https://[0000:0000:0000:0000:0000:0000:0000:0001]", "")
		assert.NoError(t, err)
		assert.Equal(t, client.environmentURL, "https://[0000:0000:0000:0000:0000:0000:0000:0001]")
	})

	t.Run("URL is without valid local path - should return an error", func(t *testing.T) {
		_, err := NewClassicClient("/my-environment/live/dynatrace.com/", "")
		assert.ErrorContains(t, err, "no host specified")
	})

	t.Run("without valid protocol - should return an error", func(t *testing.T) {
		var err error

		_, err = NewClassicClient("https//my-environment.live.dynatrace.com/", "")
		assert.ErrorContains(t, err, "not valid")

		_, err = NewClassicClient("http//my-environment.live.dynatrace.com/", "")
		assert.ErrorContains(t, err, "not valid")
	})
}

func TestNewPlatformClient(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Printf("req.URL= %+v \n", req)
		switch req.URL.Path {
		case "/oauth/token":
			token := &oauth2.Token{
				AccessToken: "test-access-token",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			}

			rw.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(rw).Encode(token)
		case classicEnvironmentDomainPath:
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte(`{"endpoint" : "/classic_endpoint"}`))
		default:
			rw.WriteHeader(404)

		}
	}))

	server.Start()
	defer server.Close()

	t.Run("Create new platform client", func(t *testing.T) {
		dtURL := server.URL

		ver := version.Version{Major: 1, Minor: 2, Patch: 3}

		c, err := NewPlatformClient(dtURL, "", OauthCredentials{TokenURL: server.URL + "/oauth/token"},
			WithServerVersion(ver),
			WithRetrySettings(rest.DefaultRetrySettings))

		assert.NoError(t, err)

		assert.NotEmpty(t, c.serverVersion)
		assert.NotEmpty(t, c.environmentURL)
		assert.NotEmpty(t, c.environmentURL)
		assert.NotEmpty(t, c.client)
		assert.NotEmpty(t, c.clientClassic)
		assert.NotEmpty(t, c.retrySettings)
		assert.NotEmpty(t, c.settingsSchemaAPIPath)
		assert.NotEmpty(t, c.settingsObjectAPIPath)

		fmt.Println(c)
		fmt.Println("")

		assert.Equal(t, dtURL, c.environmentURL)
		assert.Equal(t, "/classic_endpoint", c.environmentURLClassic)
		assert.NotEqual(t, c.client, c.clientClassic, "'classicClassic' should be same as 'client'")
		assert.Equal(t, settingsSchemaAPIPathPlatform, c.settingsSchemaAPIPath)
		assert.Equal(t, settingsObjectAPIPathPlatform, c.settingsObjectAPIPath)
		assert.Equal(t, ver, c.serverVersion, "'serverVersion' should be modified with 'WithServerVersion'")
		assert.Equal(t, rest.DefaultRetrySettings, c.retrySettings, "'retrySettings' should be modified with 'WithRetrySettings' modifier")
	})

	t.Run("URL is empty - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient(server.URL, "", OauthCredentials{TokenURL: server.URL + "/wrong/address"})
		assert.ErrorContains(t, err, "failed to query classic environment url")
	})

	t.Run("URL is empty - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("", "", OauthCredentials{})
		assert.ErrorContains(t, err, "empty url")
	})

	t.Run("invalid URL - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("INVALID_URL", "", OauthCredentials{})
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL suffix is trimmed", func(t *testing.T) {
		client, err := NewPlatformClient(server.URL, "", OauthCredentials{TokenURL: server.URL + "/oauth/token"})
		assert.NoError(t, err)
		assert.Equal(t, server.URL, client.environmentURL)
	})

	t.Run("URL with leading space - should return an error", func(t *testing.T) {
		_, err := NewPlatformClient(" https://my-environment.live.dynatrace.com/", "", OauthCredentials{})
		assert.Error(t, err)
	})

	t.Run("URL starts with http", func(t *testing.T) {
		client, err := NewPlatformClient(server.URL, "", OauthCredentials{TokenURL: server.URL + "/oauth/token"})
		assert.NoError(t, err)
		assert.Equal(t, server.URL, client.environmentURL)
	})

	t.Run("URL is without scheme - should throw an error", func(t *testing.T) {
		_, err := NewPlatformClient("my-environment.live.dynatrace.com", "", OauthCredentials{})
		assert.ErrorContains(t, err, "not valid")
	})

	t.Run("URL is without valid local path - should return an error", func(t *testing.T) {
		_, err := NewPlatformClient("/my-environment/live/dynatrace.com/", "", OauthCredentials{})
		assert.ErrorContains(t, err, "no host specified")
	})

	t.Run("without valid protocol - should return an error", func(t *testing.T) {
		var err error

		_, err = NewPlatformClient("https//my-environment.live.dynatrace.com/", "", OauthCredentials{})
		assert.ErrorContains(t, err, "not valid")

		_, err = NewPlatformClient("http//my-environment.live.dynatrace.com/", "", OauthCredentials{})
		assert.ErrorContains(t, err, "not valid")
	})
}

func TestReadByIdReturnsAnErrorUponEncounteringAnError(t *testing.T) {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		http.Error(res, "", http.StatusForbidden)
	}))
	defer func() { testServer.Close() }()
	client := DynatraceClient{
		environmentURLClassic: testServer.URL,
		clientClassic:         testServer.Client(),
	}

	_, err := client.ReadConfigById(mockAPI, "test")
	assert.ErrorContains(t, err, "Response was")
}

func TestReadByIdEscapesTheId(t *testing.T) {
	unescapedID := "ruxit.perfmon.dotnetV4:%TimeInGC:time_in_gc_alert_high_generic"

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {}))
	defer func() { testServer.Close() }()
	client := DynatraceClient{
		environmentURLClassic: testServer.URL,
		clientClassic:         testServer.Client(),
	}
	_, err := client.ReadConfigById(mockAPINotSingle, unescapedID)
	assert.NoError(t, err)
}

func TestReadByIdReturnsTheResponseGivenNoError(t *testing.T) {
	body := []byte{1, 3, 3, 7}

	testServer := httptest.NewTLSServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write(body)
	}))
	defer func() { testServer.Close() }()

	client := DynatraceClient{
		environmentURLClassic: testServer.URL,
		clientClassic:         testServer.Client(),
	}

	resp, err := client.ReadConfigById(mockAPI, "test")
	assert.NoError(t, err, "there should not be an error")
	assert.Equal(t, body, resp)
}

func TestListKnownSettings(t *testing.T) {

	tests := []struct {
		name                      string
		givenSchemaID             string
		givenListSettingsOpts     ListSettingsOptions
		givenServerResponses      []testServerResponse
		want                      []DownloadSettingsObject
		wantQueryParamsPerAPICall [][]testQueryParams
		wantNumberOfAPICalls      int
		wantError                 bool
	}{
		{
			name:          "Lists Settings objects as expected",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ] }`},
			},
			want: []DownloadSettingsObject{
				{
					ExternalId: "RG9jdG9yIFdobwo=",
					ObjectId:   "f5823eca-4838-49d0-81d9-0514dd2c4640",
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:                  "Lists Settings objects without value field as expected",
			givenSchemaID:         "builtin:something",
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
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", reducedListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Lists Settings objects with filter as expected",
			givenSchemaID: "builtin:something",
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
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Handles Pagination when listing settings objects",
			givenSchemaID: "builtin:something",
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

			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfAPICalls: 2,
			wantError:            false,
		},
		{
			name:          "Returns empty if list if no items exist",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ ] }`},
			},
			want: []DownloadSettingsObject{},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered - 400",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            true,
		},
		{
			name:          "Returns error if HTTP error is encountered - 403",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{403, `epic fail`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"schemaIds", "builtin:something"},
					{"pageSize", "500"},
					{"fields", defaultListSettingsFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            true,
		},
		{
			name:          "Retries on HTTP error on paginated request and returns eventual success",
			givenSchemaID: "builtin:something",
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
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			wantNumberOfAPICalls: 4,
			wantError:            false,
		},
		{
			name:          "Returns error if HTTP error is encountered getting further paginated responses",
			givenSchemaID: "builtin:something",
			givenServerResponses: []testServerResponse{
				{200, `{ "items": [ {"objectId": "f5823eca-4838-49d0-81d9-0514dd2c4640", "externalId": "RG9jdG9yIFdobwo="} ], "nextPageKey": "page42" }`},
				{400, `get next page fail`},
				{400, `retry fail 1`},
				{400, `retry fail 2`},
				{400, `retry fail 3`},
			},
			want: nil,
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			wantNumberOfAPICalls: 5,
			wantError:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerAPICall) > 0 {
					params := tt.wantQueryParamsPerAPICall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.NotNil(t, addedQueryParameter)
						assert.NotEmpty(t, addedQueryParameter)
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
				assert.LessOrEqualf(t, apiCalls, tt.wantNumberOfAPICalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfAPICalls, apiCalls)
			}))
			defer server.Close()

			client := DynatraceClient{
				environmentURL: server.URL,
				client:         server.Client(),
				retrySettings:  testRetrySettings,
			}

			res, err1 := client.ListSettings(tt.givenSchemaID, tt.givenListSettingsOpts)

			if tt.wantError {
				assert.Error(t, err1)
			} else {
				assert.NoError(t, err1)
			}

			assert.Equal(t, tt.want, res)

			assert.Equal(t, apiCalls, tt.wantNumberOfAPICalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfAPICalls, apiCalls)
		})
	}
}

func TestGetSettingById(t *testing.T) {
	type fields struct {
		environmentURL string
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
			if tt.fields.environmentURL != "" {
				envURL = tt.fields.environmentURL
			} else {
				envURL = server.URL
			}

			d := DynatraceClient{
				environmentURL:        envURL,
				client:                server.Client(),
				retrySettings:         tt.fields.retrySettings,
				settingsObjectAPIPath: "/api/v2/settings/objects",
			}

			settingsObj, err := d.GetSettingById(tt.args.objectID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantResult, settingsObj)

		})
	}

}

func TestDeleteSettings(t *testing.T) {
	type fields struct {
		environmentURL string
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
			if tt.fields.environmentURL != "" {
				envURL = tt.fields.environmentURL
			} else {
				envURL = server.URL
			}

			d := DynatraceClient{
				environmentURL:        envURL,
				client:                server.Client(),
				retrySettings:         tt.fields.retrySettings,
				settingsObjectAPIPath: settingsObjectAPIPathClassic,
			}

			if err := d.DeleteSettings(tt.args.objectID); (err != nil) != tt.wantErr {
				t.Errorf("DeleteSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpsertSettingsRetries(t *testing.T) {
	numAPICalls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			rw.WriteHeader(200)
			_, _ = rw.Write([]byte("{}"))
			return
		}

		numAPICalls++
		if numAPICalls < 3 {
			rw.WriteHeader(409)
			return
		}
		rw.WriteHeader(200)
		_, _ = rw.Write([]byte(`[{"objectId": "abcdefg"}]`))
	}))
	defer server.Close()

	client := DynatraceClient{
		environmentURL: server.URL,
		client:         server.Client(),
		retrySettings:  testRetrySettings,
	}

	_, err := client.UpsertSettings(SettingsObject{
		Id:       "42",
		SchemaId: "some:schema",
		Content:  []byte("{}"),
	})

	assert.NoError(t, err)
	assert.Equal(t, numAPICalls, 3)
}

func TestListEntities(t *testing.T) {

	testType := "SOMETHING"

	tests := []struct {
		name                      string
		givenEntitiesType         EntitiesType
		givenServerResponses      []testServerResponse
		want                      EntitiesList
		wantQueryParamsPerAPICall [][]testQueryParams
		wantNumberOfAPICalls      int
		wantError                 bool
	}{
		{
			name:              "Lists Entities objects as expected",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ] }`, testType, testType)},
			},
			want: EntitiesList{
				From: "",
				To:   "",
				Entities: []string{
					fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:              "Handles Pagination when listing entities objects",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-C329D7411A28B791", "type": "%s"} ] }`, testType, testType)},
			},
			want: EntitiesList{
				From: "",
				To:   "",
				Entities: []string{
					fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
					fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
				},
			},

			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
				{
					{"nextPageKey", "page42"},
				},
			},
			wantNumberOfAPICalls: 2,
			wantError:            false,
		},
		{
			name:              "Returns empty if list if no entities exist",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, `{ "entities": [ ] }`},
			},
			want: EntitiesList{
				From:     "",
				To:       "",
				Entities: []string{},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfAPICalls: 1,
			wantError:            false,
		},
		{
			name:              "Returns error if HTTP error is encountered",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{400, `epic fail`},
			},
			want: EntitiesList{},
			wantQueryParamsPerAPICall: [][]testQueryParams{
				{
					{"entitySelector", fmt.Sprintf(`type("%s")`, testType)},
					{"pageSize", defaultPageSizeEntities},
					{"fields", defaultListEntitiesFields},
				},
			},
			wantNumberOfAPICalls: 1,
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
			want: EntitiesList{
				From: "",
				To:   "",
				Entities: []string{
					fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
					fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			wantNumberOfAPICalls: 4,
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
			want: EntitiesList{},
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			wantNumberOfAPICalls: 5,
			wantError:            true,
		},
		{
			name:              "Retries on empty paginated response",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{200, fmt.Sprintf(`{ "entities": [] }`)},
				{200, fmt.Sprintf(`{ "entities": [] }`)},
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-C329D7411A28B791", "type": "%s"} ] }`, testType, testType)},
			},
			want: EntitiesList{
				From: "",
				To:   "",
				Entities: []string{
					fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
					fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			wantNumberOfAPICalls: 4,
			wantError:            false,
		},
		{
			name:              "Retries on wrong field for entity type",
			givenEntitiesType: EntitiesType{EntitiesTypeId: testType},
			givenServerResponses: []testServerResponse{
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-1A28B791C329D741", "type": "%s"} ], "nextPageKey": "page42"  }`, testType, testType)},
				{400, fmt.Sprintf(`{{
					"error":{
						"code":400,
						"message":"Constraints violated.",
						"constraintViolations":[{
							"path":"fields",
							"message":"'ipAddress' is not a valid property for type '%s'",
							"parameterLocation":"QUERY",
							"location":null
						}]
					}
				}
				}`, testType)},
				{200, fmt.Sprintf(`{ "entities": [ {"entityId": "%s-C329D7411A28B791", "type": "%s"} ] }`, testType, testType)},
			},
			want: EntitiesList{
				From: "",
				To:   "",
				Entities: []string{
					fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
					fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
				},
			},
			wantQueryParamsPerAPICall: [][]testQueryParams{
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
			},
			wantNumberOfAPICalls: 3,
			wantError:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiCalls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if len(tt.wantQueryParamsPerAPICall) > 0 {
					params := tt.wantQueryParamsPerAPICall[apiCalls]
					for _, param := range params {
						addedQueryParameter := req.URL.Query()[param.key]
						assert.NotNil(t, addedQueryParameter)
						assert.Greater(t, len(addedQueryParameter), 0)
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
				assert.LessOrEqualf(t, apiCalls, tt.wantNumberOfAPICalls, "expected at most %d API calls to happen, but encountered call %d", tt.wantNumberOfAPICalls, apiCalls)
			}))
			defer server.Close()

			client := DynatraceClient{
				environmentURL: server.URL,
				client:         server.Client(),
				retrySettings:  testRetrySettings,
			}

			res, err1 := client.ListEntities(tt.givenEntitiesType)

			if tt.wantError {
				assert.Error(t, err1)
			} else {
				assert.NoError(t, err1)
			}

			assert.Equal(t, tt.want, res)

			assert.Equal(t, apiCalls, tt.wantNumberOfAPICalls, "expected exactly %d API calls to happen but %d calls where made", tt.wantNumberOfAPICalls, apiCalls)
		})
	}
}

func TestCreateDynatraceClientWithAutoServerVersion(t *testing.T) {
	t.Run("Server version is correctly set to determined value", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(`{"version" : "1.262.0.20230214-193525"}`))
		}))

		dcl, err := NewClassicClient(server.URL, "", WithAutoServerVersion())

		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.Version{Major: 1, Minor: 262}, dcl.serverVersion)
	})

	t.Run("Server version is correctly set to unknown", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(`{}`))
		}))

		dcl, err := NewClassicClient(server.URL, "", WithAutoServerVersion())
		server.Close()
		assert.NoError(t, err)
		assert.Equal(t, version.UnknownVersion, dcl.serverVersion)
	})
}

func TestDefaultTokenURL(t *testing.T) {

	defaultTokenPath := "/fake/sso/oauth2/token"
	defaultTokenURLCalled := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		println(req)
		rw.Header().Add("Content-Type", "application/json")
		if req.URL.Path == defaultTokenPath {
			defaultTokenURLCalled = true
			rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else {
			rw.Write([]byte(`{ "some":"reply" }`))
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	defaultOAuthTokenURL = serverURL.JoinPath(defaultTokenPath).String()

	ctx := context.TODO()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, server.Client()) // ensure the oAuth client trusts the test server by passing its underlying client

	c := NewOAuthClient(ctx, OauthCredentials{
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     "", // no defined token URL should lead to default being used
	})

	_, err = c.Do(&http.Request{Method: http.MethodGet, URL: serverURL.JoinPath("/some/api/call")})
	assert.NoError(t, err)
	assert.True(t, defaultTokenURLCalled, "expected oAuth client to make an API call to the default URL")
}

func TestNonDefaultTokenURL(t *testing.T) {

	defaultTokenPath := "/fake/sso/oauth2/token"
	defaultTokenURLCalled := false

	specialTokenPath := "/magical/special/token"
	specialTokenURLCalled := false
	server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		println(req)
		rw.Header().Add("Content-Type", "application/json")
		if req.URL.Path == defaultTokenPath {
			defaultTokenURLCalled = true
			rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else if req.URL.Path == specialTokenPath {
			specialTokenURLCalled = true
			rw.Write([]byte(`{ "access_token":"ABC", "token_type":"Bearer", "expires_in":3600, "refresh_token":"ABCD", "scope":"testing" }`))
		} else {
			rw.Write([]byte(`{ "some":"reply" }`))
		}
	}))
	t.Cleanup(server.Close)

	serverURL, err := url.Parse(server.URL)
	assert.NoError(t, err)

	defaultOAuthTokenURL = serverURL.JoinPath(defaultTokenPath).String()

	ctx := context.TODO()
	ctx = context.WithValue(ctx, oauth2.HTTPClient, server.Client()) // ensure the oAuth client trusts the test server by passing its underlying client

	c := NewOAuthClient(ctx, OauthCredentials{
		ClientID:     "id",
		ClientSecret: "secret",
		TokenURL:     serverURL.JoinPath(specialTokenPath).String(),
	})

	_, err = c.Do(&http.Request{Method: http.MethodGet, URL: serverURL.JoinPath("/some/api/call")})
	assert.NoError(t, err)
	assert.True(t, specialTokenURLCalled, "expected oAuth client to make an API call to the defined token URL")
	assert.True(t, defaultTokenURLCalled == false, "expected oAuth client to make NO API call to the default URL")
}
