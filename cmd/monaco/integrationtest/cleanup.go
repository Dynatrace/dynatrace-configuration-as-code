//go:build integration || integration_v1 || cleanup || download_restore || nightly

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

package integrationtest

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/spf13/afero"
)

// CleanupIntegrationTest deletes all configs that are defined in a test manifest. It uses the CLI runner, to call the
// deletefile.Command to generate a delete file for the test manifest, and delete.GetDeleteCommand to remove configs using
// the generated file.
func CleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestPath string, specificEnvironments []string, suffix string) {

	var envArgs []string
	if len(specificEnvironments) > 0 {
		envArgs = []string{
			"--environment",
			strings.Join(specificEnvironments, ","),
		}
	}

	log.Info("### Generating delete file for test cleanup ###")

	deleteFile := fmt.Sprintf("delete-%s-%s.yaml", timeutils.TimeAnchor().UTC().Format("20060102-150405"), suffix)

	absManifestPath, err := filepath.Abs(manifestPath)
	assert.NoError(t, err)
	cmd := runner.BuildCli(fs)
	args := append([]string{
		"generate",
		"deletefile",
		absManifestPath,
		"--file", deleteFile,
		"--exclude-types", "builtin:networkzones",
	}, envArgs...)
	cmd.SetArgs(args)
	err = cmd.Execute()
	assert.NoError(t, err)

	log.Info("### Cleaning up after integration test ###")

	cmd = runner.BuildCli(fs)
	args = append([]string{
		"delete",
		"--manifest", manifestPath,
		"--file", deleteFile,
	}, envArgs...)
	cmd.SetArgs(args)
	err = cmd.Execute()

	if err != nil {
		t.Log("Failed to cleanup all test configurations, manual/nightly cleanup needed.")
	} else {
		t.Log("Successfully cleaned up test configurations.")
	}
}
