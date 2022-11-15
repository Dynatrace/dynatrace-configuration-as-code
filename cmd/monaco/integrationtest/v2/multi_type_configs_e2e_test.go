//go:build integration
// +build integration

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

package v2

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

const multiTypeProjectFolder = "test-resources/integration-multi-type-configs/"
const multiTypeManifest = multiTypeProjectFolder + "manifest.yaml"

func TestMultiTypeConfigsDeployment(t *testing.T) {

	RunIntegrationWithCleanup(t, multiTypeProjectFolder, multiTypeManifest, "", "MultiType", func(fs afero.Fs) {

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", multiTypeManifest})
		err := cmd.Execute()

		assert.NilError(t, err)

		AssertAllConfigsAvailability(t, fs, multiTypeManifest, []string{}, "", true)
	})
}

func TestMultiTypeConfigsValidation(t *testing.T) {

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", multiTypeManifest})
	err := cmd.Execute()

	assert.NilError(t, err)
}
