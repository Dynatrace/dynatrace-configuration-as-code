//go:build unit

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

package environment

import (
	"gotest.tools/assert"
	"testing"
)

func TestWithParallelRequestLimitFromEnvOption(t *testing.T) {

	t.Setenv(ConcurrentRequestsEnvKey, "")
	assert.Equal(t, DefaultConcurrentDownloads, GetEnvValueInt(ConcurrentRequestsEnvKey))
	assert.Equal(t, DefaultConcurrentDownloads, GetEnvValueIntLog(ConcurrentRequestsEnvKey))

	t.Setenv(ConcurrentRequestsEnvKey, "NOT_AN_INT")
	assert.Equal(t, DefaultConcurrentDownloads, GetEnvValueInt(ConcurrentRequestsEnvKey))
	assert.Equal(t, DefaultConcurrentDownloads, GetEnvValueIntLog(ConcurrentRequestsEnvKey))

	t.Setenv(ConcurrentRequestsEnvKey, "51")
	assert.Equal(t, 51, GetEnvValueInt(ConcurrentRequestsEnvKey))
	assert.Equal(t, 51, GetEnvValueIntLog(ConcurrentRequestsEnvKey))

	testEnvVar := "TEST_ENV_VAR_GET_ENV_VALUE"
	t.Setenv(testEnvVar, "")
	assert.Equal(t, DefaultEnvValueInt, GetEnvValueInt(testEnvVar))
	assert.Equal(t, DefaultEnvValueInt, GetEnvValueIntLog(testEnvVar))

	t.Setenv(testEnvVar, "NOT_AN_INT")
	assert.Equal(t, DefaultEnvValueInt, GetEnvValueInt(testEnvVar))
	assert.Equal(t, DefaultEnvValueInt, GetEnvValueIntLog(testEnvVar))

	t.Setenv(testEnvVar, "11")
	assert.Equal(t, 11, GetEnvValueInt(testEnvVar))
	assert.Equal(t, 11, GetEnvValueIntLog(testEnvVar))
}
