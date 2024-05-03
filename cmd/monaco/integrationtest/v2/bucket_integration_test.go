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
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/afero"
)

// Tests a dry run (validation)
func TestIntegrationBucketValidation(t *testing.T) {

	t.Setenv("UNIQUE_TEST_SUFFIX", "can-be-nonunique-for-validation")

	configFolder := "test-resources/integration-bucket/"

	t.Run("project is valid", func(t *testing.T) {
		manifest := configFolder + "manifest.yaml"
		err := monaco.Runf("monaco deploy %s --verbose --dry-run", manifest)
		assert.NoError(t, err)
	})

	t.Run("broken project is invalid", func(t *testing.T) {
		manifest := configFolder + "invalid-manifest.yaml"
		err := monaco.Runf("monaco deploy %s --verbose --dry-run", manifest)
		assert.Error(t, err)
	})
}

func TestIntegrationBucket(t *testing.T) {

	configFolder := "test-resources/integration-bucket/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "Buckets", func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.RunWithFsf(fs, "monaco deploy %s --project=project, --verbose", manifest)
		assert.NoError(t, err)

		// Update the buckets
		err = monaco.RunWithFsf(fs, "monaco deploy %s --project=project --verbose", manifest)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{"project"}, "", true)
	})
}

func TestIntegrationComplexBucket(t *testing.T) {

	configFolder := "test-resources/integration-bucket/"
	manifest := configFolder + "manifest.yaml"
	specificEnvironment := ""

	RunIntegrationWithCleanup(t, configFolder, manifest, specificEnvironment, "ComplexBuckets", func(fs afero.Fs, _ TestContext) {

		// Create the buckets
		err := monaco.RunWithFsf(fs, "monaco deploy %s --project=complex-bucket --verbose", manifest)
		assert.NoError(t, err)

		// Update the buckets
		err = monaco.RunWithFsf(fs, "monaco deploy %s --project=complex-bucket --verbose", manifest)
		assert.NoError(t, err)

		integrationtest.AssertAllConfigsAvailability(t, fs, manifest, []string{"complex-bucket"}, "", true)
	})
}
