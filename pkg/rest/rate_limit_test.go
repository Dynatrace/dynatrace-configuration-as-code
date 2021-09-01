//go:build unit
// +build unit

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
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	"strconv"
	"testing"
	"time"
)

func createTestHeaders(resetTimestamp int64) map[string][]string {

	headers := make(map[string][]string)

	headers["X-RateLimit-Limit"] = make([]string, 1)
	headers["X-RateLimit-Limit"][0] = "20"

	headers["X-RateLimit-Reset"] = make([]string, 1)
	headers["X-RateLimit-Reset"][0] = strconv.FormatInt(resetTimestamp, 10)

	return headers
}

func createTimelineProviderMock(t *testing.T) *util.MockTimelineProvider {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	return util.NewMockTimelineProvider(mockCtrl)
}

func TestDurationStaysTheSameIfInputIsWithinMinMaxLimits(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}

	value := rateLimitStrategy.applyMinMaxDefaults(6 * time.Second)
	assert.Equal(t, 6, int(value.Seconds()))
	value = rateLimitStrategy.applyMinMaxDefaults(59 * time.Second)
	assert.Equal(t, 59, int(value.Seconds()))
}

func TestDurationWillBeTheMinimumIfInputIsSmallerThanMinLimit(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}

	value := rateLimitStrategy.applyMinMaxDefaults(4 * time.Second)
	assert.Equal(t, 5, int(value.Seconds()))
	value = rateLimitStrategy.applyMinMaxDefaults(-19 * time.Second)
	assert.Equal(t, 5, int(value.Seconds()))
}

func TestDurationWillBeTheMaximumIfInputIsLargerThanMaxLimit(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}

	value := rateLimitStrategy.applyMinMaxDefaults(61 * time.Second)
	assert.Equal(t, 60, int(value.Seconds()))
	value = rateLimitStrategy.applyMinMaxDefaults(3600 * time.Second)
	assert.Equal(t, 60, int(value.Seconds()))
}

func TestRateLimitHeaderExtractionForCorrectHeaders(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	headers := createTestHeaders(0)
	response := Response{
		StatusCode: 429,
		Headers:    headers,
	}

	limit, _, resetTimeInMicroseconds, err := rateLimitStrategy.extractRateLimitHeaders(response)

	assert.NilError(t, err)
	assert.Equal(t, "20", limit)
	assert.Equal(t, 0, int(resetTimeInMicroseconds))
}

func TestRateLimitHeaderExtractionForMissingHeaders(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	response := Response{
		StatusCode: 429,
	}

	_, _, _, err := rateLimitStrategy.extractRateLimitHeaders(response)
	assert.ErrorContains(t, err, "not found")
}

func TestRateLimitHeaderExtractionForInvalidHeader(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	headers := createTestHeaders(0)
	headers["X-RateLimit-Reset"][0] = "not a unix timestamp"
	response := Response{
		StatusCode: 429,
		Headers:    headers,
	}

	_, _, _, err := rateLimitStrategy.extractRateLimitHeaders(response)
	assert.ErrorContains(t, err, "not a valid unix timestamp")
}

func TestSimpleRateLimitStrategySleepsFor42Seconds(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	timelineProvider := createTimelineProviderMock(t)
	headers := createTestHeaders(42 * time.Second.Microseconds()) // in 42 seconds
	invocationCount := 0
	callback := func() (Response, error) {

		if invocationCount == 0 {
			invocationCount++
			return Response{
				StatusCode: 429,
				Headers:    headers,
			}, nil
		}
		return Response{
			StatusCode: 200,
			Headers:    headers,
		}, nil
	}

	timelineProvider.EXPECT().Now().Times(1).Return(time.Unix(0, 0)) // time travel to the 70s
	timelineProvider.EXPECT().Sleep(42 * time.Second).Times(1)

	response, err := rateLimitStrategy.executeRequest(timelineProvider, callback)

	assert.NilError(t, err)
	assert.Equal(t, response.StatusCode, 200)
}

func TestSimpleRateLimitStrategy2Iterations(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	timelineProvider := createTimelineProviderMock(t)
	headers := createTestHeaders(42 * time.Second.Microseconds()) // in 42 seconds
	invocationCount := 0
	callback := func() (Response, error) {

		if invocationCount <= 1 {
			invocationCount++
			return Response{
				StatusCode: 429,
				Headers:    headers,
			}, nil
		}
		return Response{
			StatusCode: 200,
			Headers:    headers,
		}, nil
	}

	timelineProvider.EXPECT().Now().Times(2).Return(time.Unix(0, 0)) // time travel to the 70s
	timelineProvider.EXPECT().Sleep(42 * time.Second).Times(2)

	response, err := rateLimitStrategy.executeRequest(timelineProvider, callback)

	assert.NilError(t, err)
	assert.Equal(t, response.StatusCode, 200)
}

func TestHandleEmptyResponse(t *testing.T) {

	rateLimitStrategy := simpleSleepRateLimitStrategy{}
	timelineProvider := createTimelineProviderMock(t)
	callback := func() (Response, error) {
		return Response{}, errors.New("foo Error")
	}

	_, err := rateLimitStrategy.executeRequest(timelineProvider, callback)
	assert.ErrorContains(t, err, "foo Error")
}
