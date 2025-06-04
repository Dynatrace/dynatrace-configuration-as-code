//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package escaping

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestSpecialCharactersAreCorrectlyEscapedWhereNeeded(t *testing.T) {

	specialCharConfigFolder := "testdata/special-character-in-config/"
	specialCharManifest := filepath.Join(specialCharConfigFolder, "manifest.yaml")

	runner.Run(t, specialCharConfigFolder,
		runner.Options{
			runner.WithManifestPath(specialCharManifest),
			runner.WithSuffix("SpecialCharacterInConfig"),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", specialCharManifest))
			assert.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, specialCharManifest, []string{}, "", true)
		})
}
