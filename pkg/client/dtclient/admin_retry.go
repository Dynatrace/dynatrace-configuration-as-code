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

package dtclient

import (
	"errors"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
)

// ExecuteWithAdminAccess calls a given function with adminAccess and retries it without adminAccess in case there is a 403.
//
// returns:
//   - The response of the API call with adminAccess enabled if the permission is given
//     or the response of the API call without adminAccess if the permission is not given.
//   - Any occurring error, not related to the permission error of the adminAccess enabled call.
//   - The used adminAccess for the returned response.
func ExecuteWithAdminAccess(call func(adminAccess bool) (api.Response, error)) (api.Response, error, bool) {
	var apiErr api.APIError
	resp, err := call(true)
	if err != nil {
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
			resp, err = call(false)
			return resp, err, false
		}
		return api.Response{}, err, false
	}
	return resp, nil, true
}
