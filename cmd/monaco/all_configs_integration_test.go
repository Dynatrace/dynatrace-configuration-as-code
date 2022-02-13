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

package main

import (
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// tests all configs for a single environment
func TestIntegrationAllConfigs(t *testing.T) {

	allConfigsFolder := "test-resources/integration-all-configs/"
	allConfigsEnvironmentsFile := allConfigsFolder + "environments.yaml"

	RunIntegrationWithCleanup(t, allConfigsFolder, allConfigsEnvironmentsFile, "AllConfigs", func(fs afero.Fs) {

		// This causes a POST for all configs:
		statusCode := RunImpl([]string{
			"monaco",
			"-v",
			"--environments", allConfigsEnvironmentsFile,
			allConfigsFolder,
		}, fs)

		assert.Equal(t, statusCode, 0)

		// This causes a PUT for all configs:
		statusCode = RunImpl([]string{
			"monaco",
			"-v",
			"--environments", allConfigsEnvironmentsFile,
			// Currently there are some APIs for which updating the config does not work. These configs are included in
			// the project "only-post" (folder ./test-resources/integration-all-configs/only-post)
			// The mobile application API will be fixed in the scope of
			//     https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/issues/275
			"--project", "project",
			allConfigsFolder,
		}, fs)

		assert.Equal(t, statusCode, 0)
	})
}
