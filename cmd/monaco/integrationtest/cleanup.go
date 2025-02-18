//go:build integration || cleanup || download_restore || nightly

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
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
)

// CleanupIntegrationTest deletes all configs that are defined in a test manifest. It uses the CLI runner, to call the
// deletefile.Command to generate a delete file for the test manifest, and delete.GetDeleteCommand to remove configs using
// the generated file.
func CleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestPath string, environment string, suffix string) {
	var env string
	if len(environment) > 0 {
		env = fmt.Sprintf("--environment %s", environment)
	}

	deleteFile := fmt.Sprintf("delete-%s-%s.yaml", timeutils.TimeAnchor().UTC().Format("20060102-150405"), suffix)

	absManifestPath, err := filepath.Abs(manifestPath)
	require.NoError(t, err)

	err = monaco.RunWithFs(fs, fmt.Sprintf("monaco generate deletefile %s --file %s --exclude-types builtin:networkzones %s", absManifestPath, deleteFile, env))
	require.NoError(t, err)
	if df, err := filepath.Abs(deleteFile); err == nil {
		if b, err := afero.ReadFile(fs, df); err == nil {
			fmt.Println(string(b[:]))
		}
	}

	err = monaco.RunWithFs(fs, fmt.Sprintf("monaco --verbose delete --manifest %s --file %s %s", manifestPath, deleteFile, env))
	if err != nil {
		t.Log(err)
		t.Log("Failed to cleanup all test configurations, manual/nightly cleanup needed.")
	}
}
