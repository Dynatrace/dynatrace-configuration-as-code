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
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocuments just tries to deploy configurations containing documents and asserts whether they are indeed deployed
func TestDocuments(t *testing.T) {

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

// TestPrivateDocuments verifies that the "private" field of a document config definition in the config.yaml file
// has an effect and reaches the environment correctly.
// 1. documents are deployed (with private = true)
// 2. private is set to false for one of the documents
// 3. documents are deployed again
// 4. check whether the document is public
func TestPrivateDocuments(t *testing.T) {

	configFolder := "test-resources/integration-documents/"
	manifestPath := configFolder + "manifest.yaml"
	environment := "platform_env"

	envVars := map[string]string{featureflags.Temporary[featureflags.Documents].EnvName(): "true"}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifestPath, environment, "Documents", envVars, func(fs afero.Fs, testContext TestContext) {
		// deploy
		err := monaco.RunWithFsf(fs, "monaco deploy %s --project=project --verbose", manifestPath)
		assert.NoError(t, err)

		man, errs := manifestloader.Load(&manifestloader.Context{
			Fs:           fs,
			ManifestPath: manifestPath,
			Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
		})
		assert.Empty(t, errs)

		// check isPrivate = true for that document on environment
		clientSet := integrationtest.CreateDynatraceClients(t, man.Environments[environment])
		result, err := clientSet.Document().List(context.TODO(), fmt.Sprintf("name='my-notebook_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.True(t, result.Responses[0].IsPrivate)

		// change private field to false
		abFilePath, err := filepath.Abs(configFolder + "/project/document-notebook/config.yaml") // in monaco we are expecting files in absolute coordinates
		assert.NoError(t, err)
		file, err := fs.Open(abFilePath)
		assert.NoError(t, err)
		content, err := afero.ReadFile(fs, file.Name())
		assert.NoError(t, err)
		content = []byte(strings.ReplaceAll(string(content), "private: true", "private: false"))
		err = afero.WriteFile(fs, abFilePath, content, 0644)
		assert.NoError(t, err)

		// deploy again
		err = monaco.RunWithFsf(fs, "monaco deploy %s --project=project --verbose", manifestPath)
		assert.NoError(t, err)

		// check isPrivate = false for that document on environment
		result, err = clientSet.Document().List(context.TODO(), fmt.Sprintf("name='my-notebook_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.False(t, result.Responses[0].IsPrivate)

	})
}
