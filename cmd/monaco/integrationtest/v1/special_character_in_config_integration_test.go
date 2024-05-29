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
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSpecialCharactersAreCorrectlyEscapedWhereNeeded(t *testing.T) {

	specialCharConfigFolder := AbsOrPanicFromSlash("test-resources/special-character-in-config/")
	specialCharEnvironmentsFile := filepath.Join(specialCharConfigFolder, "environments.yaml")

	RunLegacyIntegrationWithCleanup(t, specialCharConfigFolder, specialCharEnvironmentsFile, "SpecialCharacterInConfig", func(fs afero.Fs, manifest string) {

		cmd := runner.BuildCmd(fs)
		cmd.SetArgs([]string{
			"deploy",
			"--verbose",
			manifest,
		})
		err := cmd.Execute()
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
	})
}
