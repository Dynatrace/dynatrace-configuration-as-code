//go:build integration_v1

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

package v1

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"path/filepath"
	"testing"
)

// tests all configs for a single environment
func TestIntegrationContinueDeploymentOnError(t *testing.T) {

	allConfigsFolder := AbsOrPanicFromSlash("test-resources/integration-configs-with-errors/")
	allConfigsEnvironmentsFile := filepath.Join(allConfigsFolder, "environments.yaml")

	RunLegacyIntegrationWithCleanup(t, allConfigsFolder, allConfigsEnvironmentsFile, "AllConfigs", func(fs afero.Fs, manifest string) {
		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
			"--continue-on-error",
		})
		err := cmd.Execute()
		// deployment should fail
		assert.Error(t, err, "deployment should fail")

		deployedConfig := coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard"}
		AssertConfigAvailability(t, fs, manifest, deployedConfig, "environment1", "project", true)
	})
}
