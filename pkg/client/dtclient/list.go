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
	"log/slog"
	"net/http"
	"net/url"
	"time"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/rand"
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

func listPaginated(ctx context.Context, client *corerest.Client, endpoint string, queryParams url.Values, schemaId string, addToResult AddEntriesToResult) error {
	logger := slog.With("endpoint", endpoint, "schemaId", schemaId)

	body, totalReceivedCount, err := runAndProcessResponse(ctx, client, endpoint, corerest.RequestOptions{QueryParams: queryParams, CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, addToResult)
	if err != nil {
		return err
	}

	nextPageKey, expectedTotalCount := getPaginationValues(body)
	retryCount := uint(0)
	for nextPageKey != "" {

		body, receivedCount, err := runAndProcessResponse(ctx, client, endpoint, corerest.RequestOptions{QueryParams: makeQueryParamsWithNextPageKey(endpoint, queryParams, nextPageKey), CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, addToResult)
		if err != nil {
			var apiErr coreapi.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusBadRequest {
				logger.WarnContext(ctx, "Failed to get additional data from paginated API. Pages may have been removed during request.", "response", string(body))
				break
			}
			return err
		}

		if receivedCount == 0 {
			if retryCount >= emptyResponseRetryMax {
				return fmt.Errorf("received too many empty responses (=%d)", retryCount)
			}

			retryCount++

			sleepDuration := generateSleepDuration(retryCount)
			logger.DebugContext(ctx, "Received empty array response, retrying with same 'nextPageKey'. Waiting to avoid overloading the server.", "waitDuration", sleepDuration)
			time.Sleep(sleepDuration)

			continue
		}

		retryCount = 0
		totalReceivedCount += receivedCount
		nextPageKey, _ = getPaginationValues(body)
		if nextPageKey == "" && totalReceivedCount != expectedTotalCount {
			logger.WarnContext(ctx, "Total amount of items from the API does not match with the amount of actually downloaded items.", "expectedAmount", expectedTotalCount, "receivedAmount", receivedCount, "nextPageKey", nextPageKey)
		}
	}

	return nil
}

func runAndProcessResponse(ctx context.Context, client *corerest.Client, endpoint string, requestOptions corerest.RequestOptions, addToResult AddEntriesToResult) ([]byte, int, error) {
	res, reqErr := client.GET(ctx, endpoint, requestOptions)
	if reqErr != nil {
		return nil, 0, reqErr
	}
	resp, err := coreapi.NewResponseFromHTTPResponse(res)
	if err != nil {
		return nil, 0, err
	}

	receivedCount, err := addToResult(resp.Data)
	if err != nil {
		return nil, 0, err
	}

	return resp.Data, receivedCount, nil
}

// generateSleepDuration will generate a random duration time between
//
//	1s and 1s + ([0, 1s] * backoffMultiplier)
//
// to be used between API calls.
func generateSleepDuration(backoffMultiplier uint) time.Duration {
	const backoffTime = 1 * time.Second

	waitNanos, err := rand.Int(backoffTime.Nanoseconds())
	if err != nil {
		// Since we are not reliant to cryptographically secure numbers, we can just ignore the error and assign some number.
		// It's sound enough to just wait the backoff time by default
		waitNanos = backoffTime.Nanoseconds()
	}

	return backoffTime + time.Duration(waitNanos*int64(backoffMultiplier))
}
