//go:build unit

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package schemas

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_WritesAllEmbeddedSchemas(t *testing.T) {
	outputFolder := filepath.Join(t.TempDir(), "schemas-output")

	cmd := Command(afero.NewOsFs())
	cmd.SetArgs([]string{"--output-folder", outputFolder})
	require.NoError(t, cmd.Execute())

	schemaFiles := []string{
		"monaco-account-delete-file.schema.json",
		"monaco-account-resource.schema.json",
		"monaco-config.schema.json",
		"monaco-delete-file.schema.json",
		"monaco-manifest.schema.json",
	}

	for _, name := range schemaFiles {
		t.Run(name, func(t *testing.T) {
			outPath := filepath.Join(outputFolder, name)

			_, statErr := os.Stat(outPath)
			require.NoError(t, statErr, "expected schema file %q to be written", outPath)

			written, readErr := os.ReadFile(outPath)
			require.NoError(t, readErr)

			embedded, embedErr := jsonSchemas.ReadFile("json-schemas/" + name)
			require.NoError(t, embedErr)

			assert.Equal(t, embedded, written, "content of %q does not match the embedded schema", name)
		})
	}
}
