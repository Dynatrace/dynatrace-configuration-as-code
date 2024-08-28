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
	"errors"
	"fmt"
	"net/http"
	"net/url"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/throttle"
)

const emptyResponseRetryMax = 10

// AddEntriesToResult is a function which should parse an API response body and append the returned entries to a result slice.
// Handling the parsing, any possible filtering and owning and filling the result list is left to the caller of ListPaginated,
// as it might differ notably between client implementations.
// The function MUST return the number of entries it has parsed from the received API payload body. This is used to validate
// that the final parsed number matches the reported total count of the API.
// This receivedEntries count is not necessarily equal to the number of entries added to the result slice,
// as filtering might exclude some entries that where received from the API.
type AddEntriesToResult func(body []byte) (receivedEntries int, err error)

func listPaginated(ctx context.Context, client *corerest.Client, retrySetting RetrySetting, endpoint string, queryParams url.Values, logLabel string,
	addToResult AddEntriesToResult) error {

	body, totalReceivedCount, err := runAndProcessResponse(ctx, client, retrySetting, endpoint, corerest.RequestOptions{QueryParams: queryParams, CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, addToResult)
	if err != nil {
		return err
	}

	nextPageKey, expectedTotalCount := getPaginationValues(body)
	emptyResponseRetryCount := 0
	for {
		if nextPageKey == "" {
			break
		}

		body, receivedCount, err := runAndProcessResponse(ctx, client, retrySetting, endpoint, corerest.RequestOptions{QueryParams: makeQueryParamsWithNextPageKey(endpoint, queryParams, nextPageKey), CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, addToResult)
		if err != nil {
			var apiErr coreapi.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest {
				log.Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", endpoint, string(body))
				break
			}
			return err
		}

		if receivedCount == 0 {
			if emptyResponseRetryCount >= emptyResponseRetryMax {
				return fmt.Errorf("received too many empty responses (=%d)", emptyResponseRetryCount)
			}

			emptyResponseRetryCount++
			throttle.ThrottleCallAfterError(emptyResponseRetryCount, "Received empty array response, retrying with same nextPageKey")
			continue
		}

		emptyResponseRetryCount = 0
		totalReceivedCount += receivedCount
		nextPageKey, _ = getPaginationValues(body)
		if nextPageKey == "" && totalReceivedCount != expectedTotalCount {
			log.Warn("Total count of items from api: %v for: %s does not match with count of actually downloaded items. Expected: %d Got: %d, last next page key received: %s", endpoint, logLabel, expectedTotalCount, totalReceivedCount, nextPageKey)
		}
	}

	return nil
}

func runAndProcessResponse(ctx context.Context, client *corerest.Client, retrySetting RetrySetting, endpoint string, requestOptions corerest.RequestOptions, addToResult AddEntriesToResult) ([]byte, int, error) {
	resp, err := GetWithRetry(ctx, *client, endpoint, requestOptions, retrySetting)
	if err != nil {
		return nil, 0, err
	}

	receivedCount, err := addToResult(resp.Data)
	if err != nil {
		return nil, 0, err
	}

	return resp.Data, receivedCount, nil
}
