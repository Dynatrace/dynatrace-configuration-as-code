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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	projectV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v1"
	"github.com/spf13/afero"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

func TestSpecialCharactersAreCorrectlyEscapedWhereNeeded(t *testing.T) {

	specialCharConfigFolder := AbsOrPanicFromSlash("test-resources/special-character-in-config/")
	specialCharEnvironmentsFile := filepath.Join(specialCharConfigFolder, "environments.yaml")

	RunLegacyIntegrationWithCleanup(t, specialCharConfigFolder, specialCharEnvironmentsFile, "SpecialCharacterInConfig", func(fs afero.Fs, manifest string) {

		environments, errs := environment.LoadEnvironmentList("", specialCharEnvironmentsFile, fs)
		assert.Check(t, len(errs) == 0, "didn't expect errors loading test environments")

		projects, err := projectV1.LoadProjectsToDeploy(fs, "", api.NewV1Apis(), specialCharConfigFolder)
		assert.NilError(t, err)

		cmd := runner.BuildCli(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
		})
		err = cmd.Execute()
		assert.NilError(t, err)

		AssertAllConfigsAvailability(projects, t, environments, true)
	})
}
