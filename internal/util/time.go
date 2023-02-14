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
	"fmt"
	"strconv"
	"time"
)

//go:generate mockgen -source=time.go -destination=time_mock.go -package=util TimelineProvider

// TimelineProvider abstracts away the time.Now() and time.Sleep(time.Duration) functions to make code unit-testable
// Whenever you need to get the current time, or want to pause the current goroutine (sleep), please consider using
// this interface
type TimelineProvider interface {

	// Now Returns the current (client-side) time in UTC
	Now() time.Time

	// Sleep suspends the current goroutine for the specified duration
	Sleep(duration time.Duration)
}

// NewTimelineProvider creates a new TimelineProvider
func NewTimelineProvider() TimelineProvider {
	return &defaultTimelineProvider{}
}

// defaultTimelineProvider is the default implementation of interface TimelineProvider
type defaultTimelineProvider struct{}

func (d *defaultTimelineProvider) Now() time.Time {
	nowInLocalTimeZone := time.Now()
	location, _ := time.LoadLocation("UTC")
	return nowInLocalTimeZone.In(location)
}

func (d *defaultTimelineProvider) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// StringTimestampToHumanReadableFormat parses and sanity-checks a unix timestamp as string and returns it
// as int64 and a human-readable representation of it
func StringTimestampToHumanReadableFormat(unixTimestampAsString string) (humanReadable string, parsedTimestamp int64, err error) {

	result, err := strconv.ParseInt(unixTimestampAsString, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf(
			"%s is not a valid unix timestamp",
			unixTimestampAsString,
		)
	}

	unixTimeUTC := time.Unix(result, 0)
	return unixTimeUTC.Format(time.RFC3339), result, nil
}

// ConvertMicrosecondsToUnixTime converts the UTC time in microseconds to a time.Time struct (unix time)
func ConvertMicrosecondsToUnixTime(timeInMicroseconds int64) time.Time {

	resetTimeInSeconds := timeInMicroseconds / 1000000
	resetTimeRemainderInNanoseconds := (timeInMicroseconds % 1000000) * 1000

	return time.Unix(resetTimeInSeconds, resetTimeRemainderInNanoseconds)
}
