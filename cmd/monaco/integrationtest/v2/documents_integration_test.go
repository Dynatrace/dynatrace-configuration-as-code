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
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

// TestDocuments verifies that the "private" field of a document config definition in the config.yaml file
// has an effect and reaches the environment correctly.
// 1. documents are deployed (with private = false)
// 2. private is set to true for one of the documents
// 3. documents are deployed again
// 4. check whether the document is private
func TestDocuments(t *testing.T) {

	configFolder := "test-resources/integration-documents/"
	manifestPath := configFolder + "manifest.yaml"
	environment := "platform_env"

	RunIntegrationWithCleanup(t, configFolder, manifestPath, environment, "Documents", func(fs afero.Fs, testContext TestContext) {
		// deploy
		err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=project --verbose", manifestPath))
		assert.NoError(t, err)

		man, errs := manifestloader.Load(&manifestloader.Context{
			Fs:           fs,
			ManifestPath: manifestPath,
			Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
		})
		assert.Empty(t, errs)

		// check isPrivate == false
		clientSet := integrationtest.CreateDynatraceClients(t, man.Environments[environment])
		result, err := clientSet.DocumentClient.List(t.Context(), fmt.Sprintf("name='my-notebook_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.False(t, result.Responses[0].IsPrivate)

		// check isPrivate == true
		result, err = clientSet.DocumentClient.List(t.Context(), fmt.Sprintf("name='my-dashboard_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.True(t, result.Responses[0].IsPrivate)

		// change private field to true
		abFilePath, err := filepath.Abs(configFolder + "/project/document-notebook/config.yaml") // in monaco we are expecting files in absolute coordinates
		assert.NoError(t, err)
		file, err := fs.Open(abFilePath)
		assert.NoError(t, err)
		content, err := afero.ReadFile(fs, file.Name())
		assert.NoError(t, err)
		content = []byte(strings.ReplaceAll(string(content), "private: false", "private: true"))
		err = afero.WriteFile(fs, abFilePath, content, 0644)
		assert.NoError(t, err)

		// change private field to false
		abFilePath, err = filepath.Abs(configFolder + "/project/document-dashboard/config.yaml") // in monaco we are expecting files in absolute coordinates
		assert.NoError(t, err)
		file, err = fs.Open(abFilePath)
		assert.NoError(t, err)
		content, err = afero.ReadFile(fs, file.Name())
		assert.NoError(t, err)
		content = []byte(strings.ReplaceAll(string(content), "private: true", "private: false"))
		err = afero.WriteFile(fs, abFilePath, content, 0644)
		assert.NoError(t, err)

		// deploy again
		err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=project --verbose", manifestPath))
		assert.NoError(t, err)

		// check if isPrivate was changed to true
		result, err = clientSet.DocumentClient.List(t.Context(), fmt.Sprintf("name='my-notebook_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.True(t, result.Responses[0].IsPrivate)

		// check if isPrivate was changed to false
		result, err = clientSet.DocumentClient.List(t.Context(), fmt.Sprintf("name='my-dashboard_%s'", testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 1)
		assert.False(t, result.Responses[0].IsPrivate)

		// check if both launchpads were created successfully
		result, err = clientSet.DocumentClient.List(t.Context(), fmt.Sprintf("(name='my_empty_launchpad_%s' and type='launchpad') or (name='my_monaco_launchpad_%s' and type='launchpad')", testContext.suffix, testContext.suffix))
		assert.NoError(t, err)
		assert.Len(t, result.Responses, 2)
	})
}
