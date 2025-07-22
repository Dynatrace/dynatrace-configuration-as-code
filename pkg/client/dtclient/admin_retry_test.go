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

package dtclient_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
)

func TestExecuteWithAdminAccess(t *testing.T) {
	type apiReturnValues struct {
		response api.Response
		error    error
	}

	t.Run("returns on first try with adminAccess", func(t *testing.T) {
		response := api.Response{StatusCode: http.StatusOK}
		call := func(_ bool) (api.Response, error) {
			return response, nil
		}
		resp, err, adminAccess := dtclient.ExecuteWithAdminAccess(call)
		assert.NoError(t, err)
		assert.True(t, adminAccess)
		assert.Equal(t, response, resp)
	})

	t.Run("returns on second try without adminAccess", func(t *testing.T) {
		responsesRetrySuccess := []apiReturnValues{
			{
				api.Response{},
				api.APIError{StatusCode: http.StatusForbidden},
			},
			{
				api.Response{StatusCode: 200},
				nil,
			},
		}
		calls := 0
		call := func(_ bool) (api.Response, error) {
			resp := responsesRetrySuccess[calls]
			calls++
			return resp.response, resp.error
		}
		resp, err, adminAccess := dtclient.ExecuteWithAdminAccess(call)
		assert.Equal(t, calls, 2)
		assert.NoError(t, err)
		assert.False(t, adminAccess)
		assert.Equal(t, responsesRetrySuccess[1].response, resp)
	})

	t.Run("errors if first response is not 403", func(t *testing.T) {
		respErr := api.APIError{StatusCode: http.StatusNotFound}
		call := func(_ bool) (api.Response, error) {
			return api.Response{}, respErr
		}
		_, err, adminAccess := dtclient.ExecuteWithAdminAccess(call)
		assert.Equal(t, err, respErr)
		assert.False(t, adminAccess)
	})

	t.Run("errors if second try without adminAccess errors", func(t *testing.T) {
		calls := 0
		responses := []apiReturnValues{
			{
				api.Response{},
				api.APIError{StatusCode: http.StatusForbidden},
			},
			{
				api.Response{},
				// body here just to differentiate between the other error and still have a 403 check,
				// just to be sure that sequential 403 will not lead to a loop
				api.APIError{StatusCode: http.StatusForbidden, Body: []byte("{}")},
			},
		}
		call := func(_ bool) (api.Response, error) {
			resp := responses[calls]
			calls++
			return resp.response, resp.error
		}
		_, err, adminAccess := dtclient.ExecuteWithAdminAccess(call)
		assert.Equal(t, calls, 2)
		assert.Equal(t, err, responses[1].error)
		assert.False(t, adminAccess)
	})
}
