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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
)

func createTimelineProviderMock(t *testing.T) *timeutils.MockTimelineProvider {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	return timeutils.NewMockTimelineProvider(mockCtrl)
}

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

	timelineProvider := createTimelineProviderMock(t)
	timelineProvider.EXPECT().Now().Times(100).Return(time.Unix(0, 0))

	expectedMinSleepDuration := MinWaitDuration
	expectedMaxSleepDuration := 2 * MinWaitDuration

	producedDurations := map[time.Duration]int{}
	for i := 0; i < 100; i++ {
		gotSleepDuration, _ := GenerateSleepDuration(1, timelineProvider)
		assert.Greater(t, gotSleepDuration, expectedMinSleepDuration)
		assert.LessOrEqual(t, gotSleepDuration, expectedMaxSleepDuration)

		producedDurations[gotSleepDuration] += 1
	}

	for _, times := range producedDurations {
		assert.Less(t, times, 5, "expected it less than 5% of random sleep durations to overlap")
	}
}

func TestGenerateSleepDurationSetsBackoffMultiplierOfAtLeastOne(t *testing.T) {

	timelineProvider := createTimelineProviderMock(t)
	timelineProvider.EXPECT().Now().Return(time.Unix(0, 0))

	expectedMinSleepDuration := MinWaitDuration
	expectedMaxSleepDuration := 2 * MinWaitDuration

	gotSleepDuration, _ := GenerateSleepDuration(0, timelineProvider)
	require.Greater(t, gotSleepDuration, expectedMinSleepDuration, "if backoff multiplier was >=1 sleep duration should be more than min wait")
	require.LessOrEqual(t, gotSleepDuration, expectedMaxSleepDuration)
}

func TestGenerateSleepDurationProducesHumanReadableTimestamp(t *testing.T) {

	timelineProvider := createTimelineProviderMock(t)
	timelineProvider.EXPECT().Now().Return(time.Date(2022, 10, 18, 0, 0, 0, 0, time.UTC))
	_, gotHumanReadableTimestamp := GenerateSleepDuration(1, timelineProvider)
	require.Containsf(t, gotHumanReadableTimestamp, "2022-10-18T00:00:", "expected human readable timestamp containing '2022-10-18T00:00:' but got '%s'", gotHumanReadableTimestamp)
}
