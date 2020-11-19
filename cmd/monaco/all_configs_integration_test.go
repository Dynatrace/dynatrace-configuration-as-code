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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

// tests all configs for a single environment
func TestIntegrationAllConfigs(t *testing.T) {

	const allConfigsFolder = "test-resources/integration-all-configs/"
	const allConfigsEnvironmentsFile = allConfigsFolder + "environments.yaml"

	RunIntegrationWithCleanup(t, allConfigsFolder, allConfigsEnvironmentsFile, "AllConfigs", func(fileReader util.FileReader) {

		statusCode := RunImpl([]string{
			"monaco",
			"--environments", allConfigsEnvironmentsFile,
			allConfigsFolder,
		}, fileReader)

		assert.Equal(t, statusCode, 0)
	})
}
