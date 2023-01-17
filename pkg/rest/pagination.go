// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"encoding/json"
	"net/url"
	"strings"
)

// getNextPageKeyIfExists returns the "nextPageKey" if one is found in the response body.
// This is the case for standard api/v2 pagination.
// If the response was not in the format of a paginated response, or no key was in the response an empty string is returned.
func setAdditionalValuesIfExist(response *Response) {
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(response.Body, &jsonResponse); err != nil {
		return
	}

	if jsonResponse["nextPageKey"] != nil {
		response.NextPageKey = jsonResponse["nextPageKey"].(string)
	}

	if jsonResponse["totalCount"] != nil {
		response.TotalCount = int(jsonResponse["totalCount"].(float64))
	}

	if jsonResponse["pageSize"] != nil {
		response.PageSize = int(jsonResponse["pageSize"].(float64))
	}
}

// AddNextPageQueryParams handles both Dynatrace v1 and v2 pagination logic.
// For api/v2 URLs the given next page key will be the only query parameter of the modified URL
// For any other ULRs the given next page key will be added to existing query parameters
func AddNextPageQueryParams(u *url.URL, nextPage string) *url.URL {
	queryParams := u.Query()

	if isApiV2Url(u) {
		// api/v2 requires all previously sent query params to be omitted when nextPageKey is set
		queryParams = url.Values{}
	}

	queryParams.Set("nextPageKey", nextPage)
	u.RawQuery = queryParams.Encode()
	return u
}

func isApiV2Url(url *url.URL) bool {
	return strings.Contains(url.Path, "api/v2")
}
