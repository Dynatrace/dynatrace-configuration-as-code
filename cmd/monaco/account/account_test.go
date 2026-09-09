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

package account_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

func TestEnvResolution(t *testing.T) {
	const (
		deployManifest = "testdata/multiple-accounts/manifest.yaml"
		validUUID      = "11111111-1111-1111-1111-111111111111"
	)
	fs := afero.NewOsFs()

	t.Run("validates envs of all accounts", func(t *testing.T) {
		logOutput := strings.Builder{}
		log.PrepareLogging(t.Context(), fs, false, &logOutput, false, false)

		t.Setenv("ACCOUNT_SECRET_1", validUUID)
		t.Setenv("ACCOUNT_SECRET_2", validUUID)
		t.Setenv("ACCOUNT_SECRET_URL", "https://api.dynatrace.com")

		cmd := account.Command(fs)
		cmd.SetArgs([]string{"deploy", "--dry-run", "-m", deployManifest})
		err := cmd.ExecuteContext(t.Context())

		assert.NoError(t, err)
	})

	t.Run("validates only selected account envs", func(t *testing.T) {
		t.Setenv("ACCOUNT_SECRET_1", validUUID)
		t.Setenv("ACCOUNT_SECRET_URL", "https://api.dynatrace.com")

		cmd := account.Command(fs)
		cmd.SetArgs([]string{"deploy", "-a", "monaco-test-account1", "--dry-run", "-m", deployManifest})
		err := cmd.Execute()

		assert.NoError(t, err)
	})

	t.Run("errors if selected account env is not provided", func(t *testing.T) {
		logOutput := strings.Builder{}
		log.PrepareLogging(t.Context(), fs, false, &logOutput, false, false)

		// provide env of different account
		t.Setenv("ACCOUNT_SECRET_2", validUUID)

		cmd := account.Command(fs)
		cmd.SetArgs([]string{"deploy", "-a", "monaco-test-account1", "--dry-run", "-m", deployManifest})
		err := cmd.ExecuteContext(t.Context())

		assert.Error(t, err)
		assert.Contains(t, logOutput.String(), `\"ACCOUNT_SECRET_1\" could not be found`)
	})

	t.Run("errors if envs are missing", func(t *testing.T) {
		logOutput := strings.Builder{}
		log.PrepareLogging(t.Context(), fs, false, &logOutput, false, false)

		// only one env of both is set
		t.Setenv("ACCOUNT_SECRET_1", validUUID)
		t.Setenv("ACCOUNT_SECRET_URL", "https://api.dynatrace.com")

		cmd := account.Command(fs)
		cmd.SetArgs([]string{"deploy", "--dry-run", "-m", deployManifest})
		err := cmd.ExecuteContext(t.Context())

		assert.Error(t, err)
		assert.Contains(t, logOutput.String(), `\"ACCOUNT_SECRET_2\" could not be found`)
	})
}

func TestDownloadRejectsAccountNameEscapingTheOutputFolderBeforeDownloading(t *testing.T) {
	const manifestPath = "manifest.yaml"
	const manifestContent = `manifestVersion: 1.0
projects:
- name: accounts
accounts:
- name: ../../protected-project
  accountUUID: 11111111-1111-1111-1111-111111111111
  oAuth:
    clientId:
      name: ACCOUNT_CLIENT_ID
    clientSecret:
      name: ACCOUNT_CLIENT_SECRET
`
	fs := afero.NewMemMapFs()
	require.NoError(t, afero.WriteFile(fs, manifestPath, []byte(manifestContent), 0644))
	t.Setenv("ACCOUNT_CLIENT_ID", "client-id")
	t.Setenv("ACCOUNT_CLIENT_SECRET", "client-secret")

	cmd := account.Command(fs)
	cmd.SetArgs([]string{"download", "-m", manifestPath, "--output-folder", "output"})
	err := cmd.ExecuteContext(t.Context())

	assert.EqualError(t, err, fmt.Sprintf(`cannot download account "../../protected-project": %q is not a valid project folder: it must be a relative path inside the output folder "output"`, filepath.Join("..", "protected-project")))

	outputFolderExists, statErr := afero.DirExists(fs, "output")
	require.NoError(t, statErr)
	assert.False(t, outputFolderExists)
}
