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
	"maps"
	"net/url"
	"strings"
)

// makeQueryParamsWithNextPageKey handles both Dynatrace v1 and v2 pagination logic.
// For api/v2 URLs the given next page key will be the only query parameter of the modified URL
// For any other ULRs the given next page key will be added to existing query parameters
func makeQueryParamsWithNextPageKey(endpoint string, originalQueryParams url.Values, nextPageKey string) url.Values {
	queryParams := url.Values{}

	// for non-api/v2 endpoints, we copy all original query params
	if !isApiV2Endpoint(endpoint) {
		maps.Copy(queryParams, originalQueryParams)
	}

	queryParams.Set("nextPageKey", nextPageKey)
	return queryParams
}

func isApiV2Endpoint(endpoint string) bool {
	return strings.Contains(endpoint, "api/v2")
}

func getPaginationValues(body []byte) (nextPageKey string, totalCount int) {
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		return
	}

	if jsonResponse["nextPageKey"] != nil {
		nextPageKey = jsonResponse["nextPageKey"].(string)
	}

	if jsonResponse["totalCount"] != nil {
		totalCount = int(jsonResponse["totalCount"].(float64))
	}

	return
}
