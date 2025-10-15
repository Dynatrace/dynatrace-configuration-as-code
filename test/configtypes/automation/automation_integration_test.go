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

package automation

import (
	"fmt"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	assert2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/assert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

// tests all configs for a single environment
func TestIntegrationAutomation(t *testing.T) {

	configFolder := "testdata/"
	manifest := configFolder + "manifest.yaml"

	envs := map[string]string{}
	if runner.IsHardeningEnvironment() {
		envs["WORKFLOW_ACTOR"] = os.Getenv("WORKFLOW_ACTOR")
	}

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithEnvVars(envs),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			// This causes Creation of all automation objects
			err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)

			// This causes an Update of all automation objects
			err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
			assert.NoError(t, err)

			assert2.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
		})
}
