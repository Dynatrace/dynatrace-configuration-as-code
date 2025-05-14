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

package throttle

import (
	"fmt"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/rand"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
)

const MinWaitDuration = 1 * time.Second

// ThrottleCallAfterError sleeps a bit after an error message to avoid hitting rate limits and getting the IP banned
func ThrottleCallAfterError(backoffMultiplier int, message string, a ...any) {
	timelineProvider := timeutils.NewTimelineProvider()
	sleepDuration, humanReadableTimestamp := GenerateSleepDuration(backoffMultiplier, timelineProvider)
	sleepDuration = ApplyMinMaxDefaults(sleepDuration)

	log.Debug("simpleSleepRateLimitStrategy: %s, waiting %f seconds until %s to avoid Too Many Request errors", fmt.Sprintf(message, a...), sleepDuration.Seconds(), humanReadableTimestamp)
	timelineProvider.Sleep(sleepDuration)
	log.Debug("simpleSleepRateLimitStrategy: Slept for %f seconds", sleepDuration.Seconds())
}

// GenerateSleepDuration will generate a random sleep duration time between minWaitTime and minWaitTime * backoffMultiplier
// generated sleep durations are used in case the API did not reply with a limit and reset time
// and called with the current retry iteration count to implement increasing possible wait times per iteration
func GenerateSleepDuration(backoffMultiplier int, timelineProvider timeutils.TimelineProvider) (sleepDuration time.Duration, humanReadableResetTimestamp string) {

	if backoffMultiplier < 1 {
		backoffMultiplier = 1
	}

	addedWaitMillis, err := rand.Int(MinWaitDuration.Nanoseconds())
	if err != nil {
		log.WithFields(field.Error(err)).Warn("Failed to generate random gitter. Falling back to use fixed value. Error: %s", err)
		addedWaitMillis = 0
	}

	sleepDuration = MinWaitDuration + time.Duration(addedWaitMillis*int64(backoffMultiplier))

	humanReadableResetTimestamp = timelineProvider.Now().Add(sleepDuration).Format(time.RFC3339)

	return sleepDuration, humanReadableResetTimestamp
}

func ApplyMinMaxDefaults(sleepDuration time.Duration) time.Duration {

	maxWaitTimeInNanoseconds := 1 * time.Minute

	if sleepDuration.Nanoseconds() < MinWaitDuration.Nanoseconds() {
		sleepDuration = MinWaitDuration
		log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	if sleepDuration.Nanoseconds() > maxWaitTimeInNanoseconds.Nanoseconds() {
		sleepDuration = maxWaitTimeInNanoseconds
		log.Debug("simpleSleepRateLimitStrategy: Reset sleep duration to %f seconds...", sleepDuration.Seconds())
	}
	return sleepDuration
}
