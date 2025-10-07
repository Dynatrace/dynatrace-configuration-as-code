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

package runner_test

import (
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
)

func TestRunCmd(t *testing.T) {
	fixedTime := timeutils.TimeAnchor().Format(trafficlogs.TrafficLogFilePrefixFormat)

	t.Run("it should create the support archive", func(t *testing.T) {
		fs := testutils.CreateTestFileSystem()
		cmd, supArchive := runner.BuildCmd(fs)
		cmd.SetArgs([]string{"help", "--support-archive"})
		err := runner.RunCmd(t.Context(), cmd, fs, supArchive)

		require.NoError(t, err)
		require.NotNil(t, supArchive)
		require.True(t, *supArchive)

		exists, err := afero.Exists(fs, fmt.Sprintf("support-archive-%s.zip", fixedTime))
		assert.True(t, exists)
		assert.NoError(t, err)
	})

	t.Run("it should not create the support archive", func(t *testing.T) {
		fs := testutils.CreateTestFileSystem()
		cmd, supArchive := runner.BuildCmd(fs)
		cmd.SetArgs([]string{"help"})
		err := runner.RunCmd(t.Context(), cmd, fs, supArchive)

		require.NoError(t, err)
		require.NotNil(t, supArchive)
		require.False(t, *supArchive)

		exists, err := afero.Exists(fs, fmt.Sprintf("support-archive-%s.zip", fixedTime))
		assert.False(t, exists)
		assert.NoError(t, err)
	})
}
