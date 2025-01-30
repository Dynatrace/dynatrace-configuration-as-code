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

package timeutils

import (
	"time"
)

//go:generate mockgen -source=time.go -destination=time_mock.go -package=timeutils TimelineProvider

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
