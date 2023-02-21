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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/timeutils"
	"math/rand"
	"net/http"
	"time"
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

const minWaitDuration = 1 * time.Second

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
			log.Debug("Failed to Get rate limiting details from API response, generating wait time instead...")
			sleepDuration, humanReadableTimestamp = s.generateSleepDuration(currentIteration, timelineProvider)
		}

		// That's why we need plausible min/max wait time defaults:
		sleepDuration = s.applyMinMaxDefaults(sleepDuration)

		log.Info("Rate limit reached: Applying rate limit strategy (simpleSleepRateLimitStrategy, iteration: %d)", currentIteration+1)
		log.Info("simpleSleepRateLimitStrategy: Attempting to sleep until %s", humanReadableTimestamp)

		log.Debug("simpleSleepRateLimitStrategy: Sleeping for %f seconds...", sleepDuration.Seconds())
		timelineProvider.Sleep(sleepDuration)
		log.Debug("simpleSleepRateLimitStrategy: Slept for %f seconds", sleepDuration.Seconds())

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

	log.Debug("simpleSleepRateLimitStrategy: Calculated sleep duration of %f seconds...", sleepDuration.Seconds())
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

// generateSleepDuration will generate a random sleep duration time between minWaitTime and minWaitTime * backoffMultiplier
// generated sleep durations are used in case the API did not reply with a limit and reset time
// and called with the current retry iteration count to implement increasing possible wait times per iteration
func (s *simpleSleepRateLimitStrategy) generateSleepDuration(backoffMultiplier int, timelineProvider timeutils.TimelineProvider) (sleepDuration time.Duration, humanReadableResetTimestamp string) {
	rand.Seed(time.Now().UnixNano())

	if backoffMultiplier < 1 {
		backoffMultiplier = 1
	}

	addedWaitMillis := rand.Int63n(minWaitDuration.Nanoseconds()) //nolint:gosec

	sleepDuration = minWaitDuration + time.Duration(addedWaitMillis*int64(backoffMultiplier))

	humanReadableResetTimestamp = timelineProvider.Now().Add(sleepDuration).Format(time.RFC3339)

	return sleepDuration, humanReadableResetTimestamp
}

func (s *simpleSleepRateLimitStrategy) applyMinMaxDefaults(sleepDuration time.Duration) time.Duration {

	maxWaitTimeInNanoseconds := 1 * time.Minute

	if sleepDuration.Nanoseconds() < minWaitDuration.Nanoseconds() {
		sleepDuration = minWaitDuration
		log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	if sleepDuration.Nanoseconds() > maxWaitTimeInNanoseconds.Nanoseconds() {
		sleepDuration = maxWaitTimeInNanoseconds
		log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	return sleepDuration
}
