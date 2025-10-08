/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/accounts"
	accountmanagement "github.com/dynatrace/dynatrace-configuration-as-code-core/gen/account_management"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/testutils"
)

func TestClient_GetBoundaries(t *testing.T) {

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: `{
  "pageSize": 1,
  "pageNumber": 1,
  "totalCount": 1,
  "content": [
	{
	  "uuid": "some-fake-uuid",
	  "name": "Monaco Test Boundary",
	  "boundaryQuery": "cloudautomation:event = \"helloworld\";",
	  "levelType": "account",
	  "levelId": "abcde",
	  "boundaryConditions": [],
	  "metadata": null
	}
  ]
}`,
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/boundaries?page=1&size=100", request.URL.String())
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	var instance *Client
	instance = (*Client)(accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	expected := []accountmanagement.PolicyBoundaryOverview{
		{
			Uuid:                 "some-fake-uuid",
			LevelType:            "account",
			LevelId:              "abcde",
			Name:                 "Monaco Test Boundary",
			BoundaryQuery:        "cloudautomation:event = \"helloworld\";",
			BoundaryConditions:   []accountmanagement.Condition{},
			AdditionalProperties: map[string]interface{}{},
		},
	}
	result, err := instance.GetBoundaries(t.Context(), "abcde")
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Equal(t, 1, server.Calls())
}

func TestClient_GetBoundaries_Fetch_Next_Page(t *testing.T) {

	responses := []testutils.ResponseDef{
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				content := createBoundaries(100, "abcde")
				contentJSON, _ := json.Marshal(content)
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: fmt.Sprintf(`{
  "pageSize": 100,
  "pageNumber": 1,
  "totalCount": 101,
  "content": %s
}`, contentJSON),
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/boundaries?page=1&size=100", request.URL.String())
			},
		},
		{
			GET: func(t *testing.T, request *http.Request) testutils.Response {
				content := createBoundaries(1, "abcde")
				contentJSON, _ := json.Marshal(content)
				return testutils.Response{
					ResponseCode: http.StatusOK,
					ResponseBody: fmt.Sprintf(`{
  "pageSize": 1,
  "pageNumber": 2,
  "totalCount": 1,
  "content": %s
}`, contentJSON),
				}
			},
			ValidateRequest: func(t *testing.T, request *http.Request) {
				assert.Equal(t, "/iam/v1/repo/account/abcde/boundaries?page=2&size=100", request.URL.String())
			},
		},
	}

	server := testutils.NewHTTPTestServer(t, responses)
	defer server.Close()

	var instance *Client
	instance = (*Client)(accounts.NewClient(rest.NewClient(server.URL(), server.Client())))
	boundaries, err := instance.GetBoundaries(t.Context(), "abcde")
	assert.Equal(t, 101, len(boundaries))
	assert.NoError(t, err)
	assert.Equal(t, 2, server.Calls())
}

func createBoundaries(amount int, accountUUID string) []accountmanagement.PolicyBoundaryOverview {
	var boundaries []accountmanagement.PolicyBoundaryOverview
	for i := 0; i < amount; i++ {
		boundaries = append(boundaries, accountmanagement.PolicyBoundaryOverview{
			Uuid:               fmt.Sprintf("some-fake-uuid-%d", i),
			Name:               fmt.Sprintf("some-name-%d", i),
			BoundaryQuery:      fmt.Sprintf("cloudautomation:event = \"helloworld-%d\";", i),
			LevelType:          "account",
			LevelId:            accountUUID,
			BoundaryConditions: []accountmanagement.Condition{},
			Metadata:           map[string]interface{}{},
		})
	}
	return boundaries
}
