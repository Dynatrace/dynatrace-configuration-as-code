//go:build integration

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIntegrationDocuments(t *testing.T) {

	configFolder := "test-resources/integration-documents/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	envVars := map[string]string{featureflags.Documents().EnvName(): "true"}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, specificEnvironment, "Documents", envVars, func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.RunWithFsf(fs, "monaco deploy %s --project=project --verbose", manifest)
		assert.NoError(t, err)

		// Update the buckets
		err = monaco.RunWithFsf(fs, "monaco deploy %s --project=project --verbose", manifest)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{"project"}, "", true)
	})
}
