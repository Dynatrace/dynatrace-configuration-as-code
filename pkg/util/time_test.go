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

package util

import (
	"gotest.tools/assert"
	"testing"
	"time"
)

func TestStringTimestampToHumanReadableFormatWithAValidTimestamp(t *testing.T) {

	_, parsedTimestamp, err := StringTimestampToHumanReadableFormat("0") // time travel to the 70s

	assert.NilError(t, err)
	assert.Equal(t, 0, int(parsedTimestamp))
}

func TestStringTimestampToHumanReadableFormatWithAnInvalidTimestampShouldProduceError(t *testing.T) {

	_, _, err := StringTimestampToHumanReadableFormat("abc")
	assert.ErrorContains(t, err, "is not a valid unix timestamp")
}

func TestMicrosecondsConversionToUnixTimeResultsInSameValueAfterConversion(t *testing.T) {

	unixTime := ConvertMicrosecondsToUnixTime(123456789)

	assert.Equal(t, 2, unixTime.Minute()) // 120 seconds
	assert.Equal(t, 3, unixTime.Second()) // 3 seconds (120 + 3 = 123 from above)
	assert.Equal(t, 456789000, unixTime.Nanosecond())
}

func TestTimelineProviderReturnsUTC(t *testing.T) {

	timelineProvider := NewTimelineProvider()
	now := timelineProvider.Now()

	location, _ := time.LoadLocation("UTC")
	assert.Equal(t, now.UnixNano(), now.In(location).UnixNano())
}
