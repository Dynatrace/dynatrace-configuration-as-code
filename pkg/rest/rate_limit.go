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

package rest

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/throttle"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/timeutils"
)

// rateLimitStrategy ensures that the concrete implementation of the rate limiting strategy can be hidden
// behind this interface
type rateLimitStrategy interface {
	executeRequest(timelineProvider timeutils.TimelineProvider, callback func() (Response, error)) (Response, error)
}

// createRateLimitStrategy creates a rateLimitStrategy. In the future this can be extended to instantiate
// different rate limiting strategies based on e.g. environment variables. The current implementation
// always returns the strategy simpleSleepRateLimitStrategy, which suspends the current goroutine until
// the time in the rate limiting header 'X-RateLimit-Reset' is up.
func createRateLimitStrategy() rateLimitStrategy {
	return &simpleSleepRateLimitStrategy{}
}

// simpleSleepRateLimitStrategy, is a rate limiting strategy which suspends the current goroutine until
// the time in the rate limiting header 'X-RateLimit-Reset' is up.
// It has a min sleep duration of 5 seconds and a max sleep duration of one minute and performs maximal 5
// polling iterations before giving up.
type simpleSleepRateLimitStrategy struct{}

func (s *simpleSleepRateLimitStrategy) executeRequest(timelineProvider timeutils.TimelineProvider, callback func() (Response, error)) (Response, error) {

	response, err := callback()
	if err != nil {
		return Response{}, err
	}

	maxIterationCount := 5
	currentIteration := 0

	for response.StatusCode == http.StatusTooManyRequests && currentIteration < maxIterationCount {

		sleepDuration, humanReadableTimestamp, err := s.getSleepDurationFromResponseHeader(response, timelineProvider)

		if err != nil {
			log.Debug("Failed to get rate limiting details from API response, generating wait time instead...")
			log.Debug("Response Headers: %s", response.Headers)
			log.Debug("Response Body: %s", response.Body)
			sleepDuration, humanReadableTimestamp = throttle.GenerateSleepDuration(currentIteration, timelineProvider)
		}

		// That's why we need plausible min/max wait time defaults:
		sleepDuration = throttle.ApplyMinMaxDefaults(sleepDuration)

		log.Debug("Rate limit reached (iteration: %d/%d). Sleeping until %s (%s)", currentIteration+1, maxIterationCount, humanReadableTimestamp, sleepDuration)

		timelineProvider.Sleep(sleepDuration)

		// Checking again:
		currentIteration++

		response, err = callback()
		if err != nil {
			return Response{}, err
		}
	}

	return response, nil
}

func (s *simpleSleepRateLimitStrategy) getSleepDurationFromResponseHeader(response Response, timelineProvider timeutils.TimelineProvider) (sleepDuration time.Duration, humanReadableResetTimestamp string, err error) {
	_, humanReadableTimestamp, timeInMicroseconds, err := s.extractRateLimitHeaders(response)
	if err != nil {
		return 0, "", fmt.Errorf("encountered response code 'STATUS_TOO_MANY_REQUESTS (429)' but failed to extract rate limit header: %w", err)
	}

	// Attention: this uses client time:
	now := timelineProvider.Now()

	// Attention: this uses server time:
	resetTime := timeutils.ConvertMicrosecondsToUnixTime(timeInMicroseconds)

	// Attention: this mixes client and server time:
	sleepDuration = resetTime.Sub(now)

	return sleepDuration, humanReadableTimestamp, nil
}

func (s *simpleSleepRateLimitStrategy) extractRateLimitHeaders(response Response) (limit string, humanReadableResetTimestamp string, resetTimeInMicroseconds int64, err error) {

	limitAsArray := response.Headers[http.CanonicalHeaderKey("X-RateLimit-Limit")]
	resetAsArray := response.Headers[http.CanonicalHeaderKey("X-RateLimit-Reset")]

	if limitAsArray == nil || limitAsArray[0] == "" {
		return "", "", 0, errors.New("rate limit header 'X-RateLimit-Limit' not found")
	}
	if resetAsArray == nil || resetAsArray[0] == "" {
		return "", "", 0, errors.New("rate limit header 'X-RateLimit-Reset' not found")
	}

	limit = limitAsArray[0]
	humanReadableResetTimestamp, resetTimeInMicroseconds, err = timeutils.StringTimestampToHumanReadableFormat(resetAsArray[0])
	if err != nil {
		return "", "", 0, err
	}

	return limit, humanReadableResetTimestamp, resetTimeInMicroseconds, nil
}
