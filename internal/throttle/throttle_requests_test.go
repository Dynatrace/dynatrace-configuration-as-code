//go:build unit || integration || download_restore || cleanup || nightly

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
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationStaysTheSameIfInputIsWithinMinMaxLimits(t *testing.T) {

	value := ApplyMinMaxDefaults(6 * time.Second)
	require.Equal(t, 6, int(value.Seconds()))
	value = ApplyMinMaxDefaults(59 * time.Second)
	require.Equal(t, 59, int(value.Seconds()))
}

func TestDurationWillBeTheMinimumIfInputIsSmallerThanMinLimit(t *testing.T) {

	value := ApplyMinMaxDefaults(500 * time.Millisecond)
	require.Equal(t, 1, int(value.Seconds()))
	value = ApplyMinMaxDefaults(-19 * time.Second)
	require.Equal(t, 1, int(value.Seconds()))
}

func TestDurationWillBeTheMaximumIfInputIsLargerThanMaxLimit(t *testing.T) {

	value := ApplyMinMaxDefaults(61 * time.Second)
	require.Equal(t, 60, int(value.Seconds()))
	value = ApplyMinMaxDefaults(3600 * time.Second)
	require.Equal(t, 60, int(value.Seconds()))
}

func TestGeneratedSleepDurationsAreWithinExpectedBoundsAndDistribution(t *testing.T) {

	expectedMinSleepDuration := MinWaitDuration
	expectedMaxSleepDuration := 2 * MinWaitDuration

	producedDurations := map[time.Duration]int{}
	for i := 0; i < 100; i++ {
		gotSleepDuration, _ := GenerateSleepDuration(1)
		assert.Greater(t, gotSleepDuration, expectedMinSleepDuration)
		assert.LessOrEqual(t, gotSleepDuration, expectedMaxSleepDuration)

		producedDurations[gotSleepDuration] += 1
	}

	for _, times := range producedDurations {
		assert.Less(t, times, 5, "expected it less than 5% of random sleep durations to overlap")
	}
}

func TestGenerateSleepDurationSetsBackoffMultiplierOfAtLeastOne(t *testing.T) {
	expectedMinSleepDuration := MinWaitDuration
	expectedMaxSleepDuration := 2 * MinWaitDuration

	synctest.Run(func() {
		gotSleepDuration, _ := GenerateSleepDuration(0)
		require.Greater(t, gotSleepDuration, expectedMinSleepDuration, "if backoff multiplier was >=1 sleep duration should be more than min wait")
		require.LessOrEqual(t, gotSleepDuration, expectedMaxSleepDuration)

	})

}

func TestGenerateSleepDurationProducesHumanReadableTimestamp(t *testing.T) {
	synctest.Run(func() {
		nowFormatted := time.Now().UTC().Format(time.RFC3339)

		_, gotHumanReadableTimestamp := GenerateSleepDuration(1)
		require.Contains(t, gotHumanReadableTimestamp, nowFormatted, "expected human readable timestamp containing timestamp")
	})
}

func TestThrottleCallAfterError(t *testing.T) {
	expectedMinSleepDuration := MinWaitDuration
	expectedMaxSleepDuration := 2 * MinWaitDuration

	synctest.Run(func() {
		start := time.Now().UTC()

		ThrottleCallAfterError(1, "test")

		after := time.Now().UTC()

		require.Greater(t, after, start.Add(expectedMinSleepDuration))
		require.LessOrEqual(t, after, start.Add(expectedMaxSleepDuration))
	})
}
