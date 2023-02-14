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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// tests all configs for a single environment
func TestIntegrationAllConfigs(t *testing.T) {

	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "AllConfigs", func(fs afero.Fs) {

		// This causes a POST for all configs:

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err := cmd.Execute()

		assert.NilError(t, err)

		// This causes a PUT for all configs:

		cmd = runner.BuildCli(fs)
		cmd.SetArgs([]string{"deploy", "--verbose", manifest})
		err = cmd.Execute()
		assert.NilError(t, err)

	})
}

// Tests a dry run (validation)
func TestIntegrationValidationAllConfigs(t *testing.T) {

	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	cmd := runner.BuildCli(util.CreateTestFileSystem())
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.NilError(t, err)
}
