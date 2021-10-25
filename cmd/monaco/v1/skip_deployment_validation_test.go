//go:build integration_v1
// +build integration_v1

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
	"testing"

	"gotest.tools/assert"

	"fmt"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/main/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/envvars"
)

var skipDeploymentFolder = AbsOrPanicFromSlash("test-resources/skip-deployment-project/")
var skipDeploymentEnvironmentsFile = AbsOrPanicFromSlash("test-resources/test-environments.yaml")

func TestValidationSkipDeployment(t *testing.T) {
	envvars.InstallFakeEnvironment(map[string]string{
		"CONFIG_V1": "1",
	})

	defer envvars.InstallOsBased()

	statusCode := runner.RunImpl([]string{
		"monaco",
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectA",
		skipDeploymentFolder,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

func TestValidationSkipDeploymentWithBrokenDependency(t *testing.T) {
	envvars.InstallFakeEnvironment(map[string]string{
		"CONFIG_V1": "1",
	})

	defer envvars.InstallOsBased()

	statusCode := runner.RunImpl([]string{
		"monaco",
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectB",
		skipDeploymentFolder,
	}, util.CreateTestFileSystem())

	assert.Assert(t, statusCode != 0, fmt.Sprintf("Status code (%d) should be error", statusCode))
}

func TestValidationSkipDeploymentWithOverridingDependency(t *testing.T) {
	envvars.InstallFakeEnvironment(map[string]string{
		"CONFIG_V1": "1",
	})

	defer envvars.InstallOsBased()

	statusCode := runner.RunImpl([]string{
		"monaco",
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectC",
		skipDeploymentFolder,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

func TestValidationSkipDeploymentWithOverridingFlagValue(t *testing.T) {
	envvars.InstallFakeEnvironment(map[string]string{
		"CONFIG_V1": "1",
	})

	defer envvars.InstallOsBased()

	statusCode := runner.RunImpl([]string{
		"monaco",
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectE",
		skipDeploymentFolder,
	}, util.CreateTestFileSystem())

	assert.Equal(t, statusCode, 0)
}

func TestValidationSkipDeploymentInterProjectWithMissingDependency(t *testing.T) {
	envvars.InstallFakeEnvironment(map[string]string{
		"CONFIG_V1": "1",
	})

	defer envvars.InstallOsBased()

	statusCode := runner.RunImpl([]string{
		"monaco",
		"deploy",
		"--verbose",
		"--dry-run",
		"--environments", skipDeploymentEnvironmentsFile,
		"--project", "projectD",
		skipDeploymentFolder,
	}, util.CreateTestFileSystem())

	assert.Assert(t, statusCode != 0)
}
