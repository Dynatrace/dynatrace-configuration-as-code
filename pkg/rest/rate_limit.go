/**
 * @license
 * Copyright 2020 Dynatrace LLC
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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"net/http"
	"time"
)

// rateLimitStrategy ensures that the concrete implementation of the rate limiting strategy can be hidden
// behind this interface
type rateLimitStrategy interface {
	executeRequest(timelineProvider util.TimelineProvider, callback func() (Response, error)) (Response, error)
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

func (s *simpleSleepRateLimitStrategy) executeRequest(timelineProvider util.TimelineProvider, callback func() (Response, error)) (Response, error) {

	response, err := callback()
	if err != nil {
		return Response{}, err
	}

	maxIterationCount := 5
	currentIteration := 0

	for response.StatusCode == http.StatusTooManyRequests && currentIteration < maxIterationCount {

		limit, humanReadableTimestamp, timeInMicroseconds, err := s.extractRateLimitHeaders(response)
		if err != nil {
			return response, err
		}

		util.Log.Info("Rate limit of %d requests/min reached: Applying rate limit strategy (simpleSleepRateLimitStrategy, iteration: %d)", limit, currentIteration+1)
		util.Log.Info("simpleSleepRateLimitStrategy: Attempting to sleep until %s", humanReadableTimestamp)

		// Attention: this uses client time:
		now := timelineProvider.Now()

		// Attention: this uses server time:
		resetTime := util.ConvertMicrosecondsToUnixTime(timeInMicroseconds)

		// Attention: this mixes client and server time:
		sleepDuration := resetTime.Sub(now)
		util.Log.Debug("simpleSleepRateLimitStrategy: Calculated sleep duration of %f seconds...", sleepDuration.Seconds())

		// That's why we need plausible min/max wait time defaults:
		sleepDuration = s.applyMinMaxDefaults(sleepDuration)

		util.Log.Debug("simpleSleepRateLimitStrategy: Sleeping for %f seconds...", sleepDuration.Seconds())
		timelineProvider.Sleep(sleepDuration)
		util.Log.Debug("simpleSleepRateLimitStrategy: Slept for %f seconds", sleepDuration.Seconds())

		// Checking again:
		currentIteration++

		response, err = callback()
		if err != nil {
			return Response{}, err
		}
	}

	return response, nil
}

func (s *simpleSleepRateLimitStrategy) extractRateLimitHeaders(response Response) (limit string, humanReadableResetTimestamp string, resetTimeInMicroseconds int64, err error) {

	limitAsArray := response.Headers["X-RateLimit-Limit"]
	resetAsArray := response.Headers["X-RateLimit-Reset"]

	if limitAsArray == nil || limitAsArray[0] == "" {
		return "", "", 0, errors.New("rate limit header 'X-RateLimit-Limit' not found")
	}
	if resetAsArray == nil || resetAsArray[0] == "" {
		return "", "", 0, errors.New("rate limit header 'X-RateLimit-Reset' not found")
	}

	limit = limitAsArray[0]
	humanReadableResetTimestamp, resetTimeInMicroseconds, err = util.StringTimestampToHumanReadableFormat(resetAsArray[0])
	if err != nil {
		return "", "", 0, err
	}

	return limit, humanReadableResetTimestamp, resetTimeInMicroseconds, nil
}

func (s *simpleSleepRateLimitStrategy) applyMinMaxDefaults(sleepDuration time.Duration) time.Duration {

	minWaitTimeInNanoseconds := 5 * time.Second
	maxWaitTimeInNanoseconds := 1 * time.Minute

	if sleepDuration.Nanoseconds() < minWaitTimeInNanoseconds.Nanoseconds() {
		sleepDuration = minWaitTimeInNanoseconds
		util.Log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	if sleepDuration.Nanoseconds() > maxWaitTimeInNanoseconds.Nanoseconds() {
		sleepDuration = maxWaitTimeInNanoseconds
		util.Log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	return sleepDuration
}
