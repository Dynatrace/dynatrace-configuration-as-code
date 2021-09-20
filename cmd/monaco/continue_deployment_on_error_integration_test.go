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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/spf13/afero"

	"gotest.tools/assert"
)

// tests all configs for a single environment
func TestIntegrationContinueDeploymentOnError(t *testing.T) {

	const allConfigsFolder = "test-resources/integration-configs-with-errors/"
	const allConfigsEnvironmentsFile = allConfigsFolder + "environments.yaml"

	RunIntegrationWithCleanup(t, allConfigsFolder, allConfigsEnvironmentsFile, "AllConfigs", func(fs afero.Fs) {

		environments, errs := environment.LoadEnvironmentList("", allConfigsEnvironmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := project.LoadProjectsToDeploy(fs, "", api.NewApis(), allConfigsFolder)
		assert.NilError(t, err)

		statusCode := RunImpl([]string{
			"monaco",
			"--verbose", "--continue-on-error",
			"--environments", allConfigsEnvironmentsFile,
			allConfigsFolder,
		}, fs)

		// deployment should fail
		assert.Equal(t, statusCode, 1)

		// dashboard should anyways be deployed
		dashboardConfig, err := projects[0].GetConfig("test-resources/integration-configs-with-errors/project/dashboard/dashboard")
		assert.NilError(t, err)
		AssertConfigAvailability(t, dashboardConfig, environments["environment1"], true)
	})
}
