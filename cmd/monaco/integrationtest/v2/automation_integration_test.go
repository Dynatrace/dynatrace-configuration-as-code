//go:build integration

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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
	"fmt"
	"os"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// tests all configs for a single environment
func TestIntegrationAutomation(t *testing.T) {

	configFolder := "test-resources/integration-automation/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	envs := map[string]string{}
	if isHardeningEnvironment() {
		envs["WORKFLOW_ACTOR"] = os.Getenv("WORKFLOW_ACTOR")
	}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, specificEnvironment, "Automation", envs, func(fs afero.Fs, _ TestContext) {
		// This causes Creation of all automation objects
		err := monaco.RunWithFs(fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
		assert.NoError(t, err)

		// This causes an Update of all automation objects
		err = monaco.RunWithFs(fs, fmt.Sprintf("monaco deploy %s --verbose", manifest))
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{}, "", true)
	})
}
