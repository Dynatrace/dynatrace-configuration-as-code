//go:build unit

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package supportarchive_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
)

func TestIsEnabled(t *testing.T) {
	t.Run("If support archive is set, it's enabled", func(t *testing.T) {
		ctx := supportarchive.ContextWithSupportArchive(t.Context())

		assert.True(t, supportarchive.IsEnabled(ctx))
	})

	t.Run("If support archive isn't set, it's disabled", func(t *testing.T) {
		assert.False(t, supportarchive.IsEnabled(t.Context()))
	})
}

func TestWrite(t *testing.T) {
	fixedTime := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat)
	fs := testutils.CreateTestFileSystem()

	t.Run("It should not error if response and request files are missing", func(t *testing.T) {
		logFilePath := fixedTime + ".log"
		errorFilePath := fixedTime + "-errors.log"

		preExistingFiles := []string{
			logFilePath,
			errorFilePath,
		}

		err := fs.Mkdir(log.LogDirectory, 0644)
		require.NoError(t, err)

		for _, file := range preExistingFiles {
			f, err := fs.Create(path.Join(log.LogDirectory, file))
			require.NoError(t, err)
			err = f.Close()
			require.NoError(t, err)
		}

		err = supportarchive.Write(fs)
		require.NoError(t, err)

		archive := fmt.Sprintf("support-archive-%s.zip", fixedTime)
		expectedFiles := []string{
			logFilePath,
			errorFilePath,
			fixedTime + "-featureflag_state.log",
		}

		testutils.AssertSupportArchive(t, fs, archive, expectedFiles)
	})

	t.Run("It should contain all files", func(t *testing.T) {
		reportFileName := fixedTime + "-report.jsonl"
		logFilePath := fixedTime + ".log"
		errorFilePath := fixedTime + "-errors.log"
		requestFilePath := fixedTime + "-req.log"
		responseFilePath := fixedTime + "-resp.log"
		memStatFilePath := fixedTime + "-memstat.log"

		t.Setenv(featureflags.LogMemStats.EnvName(), "true")
		t.Setenv(environment.DeploymentReportFilename, path.Join(log.LogDirectory, reportFileName))

		preExistingFiles := []string{
			logFilePath,
			errorFilePath,
			requestFilePath,
			responseFilePath,
			memStatFilePath,
			reportFileName,
		}

		err := fs.Mkdir(log.LogDirectory, 0644)
		require.NoError(t, err)

		for _, file := range preExistingFiles {
			f, err := fs.Create(path.Join(log.LogDirectory, file))
			require.NoError(t, err)
			err = f.Close()
			require.NoError(t, err)
		}

		expectedFiles := []string{
			requestFilePath,
			responseFilePath,
			logFilePath,
			errorFilePath,
			reportFileName,
			memStatFilePath,
			fixedTime + "-featureflag_state.log",
		}

		err = supportarchive.Write(fs)
		require.NoError(t, err)
		archive := fmt.Sprintf("support-archive-%s.zip", fixedTime)

		testutils.AssertSupportArchive(t, fs, archive, expectedFiles)
	})
}
