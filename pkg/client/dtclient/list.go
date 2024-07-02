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
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/throttle"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
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

func listPaginated(ctx context.Context, client *rest.Client, retrySettings rest.RetrySettings, url *url.URL, logLabel string,
	addToResult AddEntriesToResult) (rest.Response, error) {

	var resp rest.Response
	startTime := time.Now()
	receivedCount := 0
	totalReceivedCount := 0

	resp, receivedCount, totalReceivedCount, _, err := runAndProcessResponse(ctx, client, retrySettings, false, url, addToResult, receivedCount, totalReceivedCount)
	if err != nil {
		return buildResponseError(err, resp, url)
	}

	nbCalls := 1
	lastLogTime := time.Now()
	expectedTotalCount := resp.TotalCount
	nextPageKey := resp.NextPageKey
	emptyResponseRetryCount := 0

	for {

		if nextPageKey != "" {
			logLongRunningExtractionProgress(&lastLogTime, startTime, nbCalls, resp, logLabel)

			url = addNextPageQueryParams(url, nextPageKey)

			var isLastAvailablePage bool
			resp, receivedCount, totalReceivedCount, isLastAvailablePage, err = runAndProcessResponse(ctx, client, retrySettings, true, url, addToResult, receivedCount, totalReceivedCount)
			if err != nil {
				return buildResponseError(err, resp, url)
			}
			if isLastAvailablePage {
				break
			}

			retry := false
			retry, emptyResponseRetryCount, err = isRetryOnEmptyResponse(receivedCount, emptyResponseRetryCount, resp)
			if err != nil {
				return buildResponseError(err, resp, url)
			}

			if retry {
				continue
			} else {
				validateWrongCountExtracted(resp, totalReceivedCount, expectedTotalCount, url, logLabel, nextPageKey)

				nextPageKey = resp.NextPageKey
				nbCalls++
				emptyResponseRetryCount = 0
			}

		} else {

			break
		}
	}

	return resp, nil
}

func logLongRunningExtractionProgress(lastLogTime *time.Time, startTime time.Time, nbCalls int, resp rest.Response, logLabel string) {
	if time.Since(*lastLogTime).Minutes() >= 1 {
		*lastLogTime = time.Now()
		nbItemsMessage := ""
		ETAMessage := ""
		runningMinutes := time.Since(startTime).Minutes()
		nbCallsPerMinute := float64(nbCalls) / runningMinutes
		if resp.PageSize > 0 && resp.TotalCount > 0 {
			nbProcessed := nbCalls * resp.PageSize
			nbLeft := resp.TotalCount - nbProcessed
			ETAMinutes := float64(nbLeft) / (nbCallsPerMinute * float64(resp.PageSize))
			nbItemsMessage = fmt.Sprintf(", processed %d of %d at %d items/call and", nbProcessed, resp.TotalCount, resp.PageSize)
			ETAMessage = fmt.Sprintf("ETA: %.1f minutes", ETAMinutes)
		}

		log.Debug("Running extraction of: %s for %.1f minutes%s %.1f call/minute. %s", logLabel, runningMinutes, nbItemsMessage, nbCallsPerMinute, ETAMessage)
	}
}

func validateWrongCountExtracted(resp rest.Response, totalReceivedCount int, expectedTotalCount int, url *url.URL, logLabel string, nextPageKey string) {
	if resp.NextPageKey == "" && totalReceivedCount != expectedTotalCount {
		log.Warn("Total count of items from api: %v for: %s does not match with count of actually downloaded items. Expected: %d Got: %d, last next page key received: %s \n   params: %v", url.Path, logLabel, expectedTotalCount, totalReceivedCount, nextPageKey, url.RawQuery)
	}
}

func isRetryOnEmptyResponse(receivedCount int, emptyResponseRetryCount int, resp rest.Response) (bool, int, error) {
	if receivedCount == 0 {
		if emptyResponseRetryCount < emptyResponseRetryMax {
			emptyResponseRetryCount++
			throttle.ThrottleCallAfterError(emptyResponseRetryCount, "Received empty array response, retrying with same nextPageKey (HTTP: %d) ", resp.StatusCode)
			return true, emptyResponseRetryCount, nil
		} else {
			return false, emptyResponseRetryCount, fmt.Errorf("received too many empty responses (=%d)", emptyResponseRetryCount)
		}
	}

	return false, emptyResponseRetryCount, nil
}

func runAndProcessResponse(ctx context.Context, client *rest.Client, retrySettings rest.RetrySettings, isNextCall bool, u *url.URL,
	addToResult AddEntriesToResult, receivedCount int, totalReceivedCount int) (rest.Response, int, int, bool, error) {
	isLastAvailablePage := false

	resp, err := client.GetWithRetry(ctx, u.String(), retrySettings.Normal)
	isLastAvailablePage, err = validateRespErrors(isNextCall, err, resp, u.Path)
	if err != nil || isLastAvailablePage {
		return resp, receivedCount, totalReceivedCount, isLastAvailablePage, err
	}

	receivedCount, err = addToResult(resp.Body)
	totalReceivedCount += receivedCount

	return resp, receivedCount, totalReceivedCount, isLastAvailablePage, err
}

func validateRespErrors(isNextCall bool, err error, resp rest.Response, urlPath string) (bool, error) {
	if err != nil {
		return false, err
	}
	isLastAvailablePage := false
	if resp.IsSuccess() {
		return false, nil

	} else if isNextCall {
		if resp.StatusCode == http.StatusBadRequest {
			isLastAvailablePage = true
			log.Warn("Failed to get additional data from paginated API %s - pages may have been removed during request.\n    Response was: %s", urlPath, string(resp.Body))
			return isLastAvailablePage, nil
		} else {
			return isLastAvailablePage, fmt.Errorf("failed to get further data from paginated API %s (HTTP %d)!\n    Response was: %s", urlPath, resp.StatusCode, string(resp.Body))
		}
	} else {
		return isLastAvailablePage, fmt.Errorf("failed to get data from paginated API %s (HTTP %d)!\n    Response was: %s", urlPath, resp.StatusCode, string(resp.Body))
	}

}

func buildResponseError(err error, resp rest.Response, url *url.URL) (rest.Response, error) {
	var respErr rest.RespError
	if errors.As(err, &respErr) {
		return resp, fmt.Errorf("failed to process paginated API response: %w", respErr)
	}
	return resp, rest.NewRespErr("failed to process paginated API response", resp).WithRequestInfo(http.MethodGet, url.String()).WithErr(err)
}
