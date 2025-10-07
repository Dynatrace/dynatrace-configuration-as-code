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

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
)

func TestExecuteMain(t *testing.T) {
	fixedTime := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat)

	t.Run("It should gracefully execute the main program", func(t *testing.T) {
		supportArchive := fmt.Sprintf("support-archive-%s.zip", fixedTime)
		logFilePath := fixedTime + ".log"
		errorFilePath := fixedTime + "-errors.log"
		expectedFiles := []string{
			logFilePath,
			errorFilePath,
			fixedTime + "-featureflag_state.log",
		}
		fs := testutils.CreateTestFileSystem()

		os.Args = []string{os.Args[0], "help", "--support-archive"}
		exitCode := executeMain(fs)
		require.Zero(t, exitCode)

		testutils.AssertSupportArchive(t, fs, supportArchive, expectedFiles)
	})

	t.Run("It shutdown the main program", func(t *testing.T) {
		supportArchive := fmt.Sprintf("support-archive-%s.zip", fixedTime)
		fs := testutils.CreateTestFileSystem()

		os.Args = []string{os.Args[0], "deploy", "not-existing-manifest.yaml", "--support-archive"}
		exitCode := executeMain(fs)
		require.Equal(t, 1, exitCode)

		testutils.AssertSupportArchiveContainsError(t, fs, supportArchive, "error while loading manifest")
	})
}
