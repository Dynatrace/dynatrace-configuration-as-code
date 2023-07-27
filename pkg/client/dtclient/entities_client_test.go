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
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/concurrency"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_fields(t *testing.T) {
	APMSecurityGatewayJSON := `{
		"type": "APM_SECURITY_GATEWAY",
		"displayName": "ActiveGate",
		"dimensionKey": "dt.entity.apm_security_gateway",
		"entityLimitExceeded": false,
		"properties": [
			{"id": "awsNameTag", "type": "String", "displayName": "awsNameTag"},
			{"id": "boshName", "type": "String", "displayName": "boshName"},
			{"id": "conditionalName", "type": "String", "displayName": "conditionalName"},
			{"id": "customizedName", "type": "String", "displayName": "customizedName"},
			{"id": "detectedName", "type": "String", "displayName": "detectedName"},
			{"id": "gcpZone", "type": "String", "displayName": "gcpZone"},
			{"id": "isContainerDeployment", "type": "Boolean", "displayName": "isContainerDeployment"},
			{"id": "oneAgentCustomHostName", "type": "String", "displayName": "oneAgentCustomHostName"}
		],
		"tags": "List",
		"managementZones": "List",
		"fromRelationships": [
			{"id": "isLocatedIn", "toTypes": ["SYNTHETIC_LOCATION"]}
		],
		"toRelationships": []
	}`

	tests := []struct {
		name             string
		EntitiesTypeJSON string
		ignoreProperties []string
		want             string
	}{
		{
			"Extract Values - without ignore",
			APMSecurityGatewayJSON,
			[]string{},
			"+lastSeenTms,+firstSeenTms,+properties.detectedName,+properties.oneAgentCustomHostName",
		},
		{
			"Extract Values - ignore 1 property",
			APMSecurityGatewayJSON,
			[]string{"detectedName"},
			"+lastSeenTms,+firstSeenTms,+properties.oneAgentCustomHostName",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			entitiesType := EntitiesType{}
			json.Unmarshal([]byte(tt.EntitiesTypeJSON), &entitiesType)
			fields := getEntitiesTypeFields(entitiesType, tt.ignoreProperties)
			assert.Equal(t, fields, tt.want)
		})
	}
}

func TestListEntities(t *testing.T) {

	testType := "SOMETHING"

	tests := []struct {
		name                      string
		givenEntitiesType         EntitiesType
		givenServerResponses      []testServerResponse
		want                      []string
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
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
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
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
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
			want: []string{},
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
			want: nil,
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
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
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
			want: nil,
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
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
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
			want: []string{
				fmt.Sprintf(`{"entityId": "%s-1A28B791C329D741", "type": "%s"}`, testType, testType),
				fmt.Sprintf(`{"entityId": "%s-C329D7411A28B791", "type": "%s"}`, testType, testType),
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
				environmentURL:     server.URL,
				platformClient:     rest.NewRestClient(server.Client(), nil, rest.CreateRateLimitStrategy()),
				retrySettings:      testRetrySettings,
				limiter:            concurrency.NewLimiter(5),
				generateExternalID: idutils.GenerateExternalID,
			}

			res, err1 := client.ListEntities(context.TODO(), tt.givenEntitiesType)

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
