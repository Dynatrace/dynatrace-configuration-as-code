//go:build unit

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

package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithParallelRequestLimitFromEnvOption(t *testing.T) {

	testEnvVar := ConcurrentRequestsEnvKey
	t.Setenv(ConcurrentRequestsEnvKey, "")
	require.Equal(t, defaultValuesInt[testEnvVar], GetEnvValueInt(testEnvVar), "expected default value if no env var is set")
	require.Equal(t, defaultValuesInt[testEnvVar], GetEnvValueIntLog(testEnvVar), "expected default value if no env var is set")
	require.Equal(t, "Concurrent Request Limit: %d, '%s' environment variable is NOT set, using default value", getLogMessage(testEnvVar, logStringIntDefault), "expected default value if no env var is set")

	t.Setenv(testEnvVar, "NOT_AN_INT")
	require.Equal(t, defaultValuesInt[testEnvVar], GetEnvValueInt(testEnvVar), "expected default value if env var is not an integer")
	require.Equal(t, defaultValuesInt[testEnvVar], GetEnvValueIntLog(testEnvVar), "expected default value if env var is not an integer")
	require.Equal(t, "Concurrent Request Limit: %d, '%s' environment variable is NOT set, using default value", getLogMessage(testEnvVar, logStringIntDefault), "expected default value if env var is not an integer")

	t.Setenv(testEnvVar, "51")
	require.Equal(t, 51, GetEnvValueInt(testEnvVar))
	require.Equal(t, 51, GetEnvValueIntLog(testEnvVar))
	require.Equal(t, "Concurrent Request Limit: %d, from '%s' environment variable", getLogMessage(testEnvVar, logStringInt))

	testEnvVar = "TEST_ENV_VAR_GET_ENV_VALUE"
	t.Setenv(testEnvVar, "")
	require.Equal(t, 0, GetEnvValueInt(testEnvVar))
	require.Equal(t, 0, GetEnvValueIntLog(testEnvVar))
	require.Equal(t, "Environment variable %s: %d, variable is NOT set, using default value", getLogMessage(testEnvVar, logStringIntDefault))

	t.Setenv(testEnvVar, "NOT_AN_INT")
	require.Equal(t, 0, GetEnvValueInt(testEnvVar))
	require.Equal(t, 0, GetEnvValueIntLog(testEnvVar))
	require.Equal(t, "Environment variable %s: %d, variable is NOT set, using default value", getLogMessage(testEnvVar, logStringIntDefault))

	t.Setenv(testEnvVar, "11")
	require.Equal(t, 11, GetEnvValueInt(testEnvVar))
	require.Equal(t, 11, GetEnvValueIntLog(testEnvVar))
	require.Equal(t, "Environment variable %s: %d", getLogMessage(testEnvVar, logStringInt))
}
