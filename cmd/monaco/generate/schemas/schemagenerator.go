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

package schemas

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	accountDelete "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
	account "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence"
	config "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/writer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	manifest "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/writer"
)

func generateSchemaFiles(fs afero.Fs, outputfolder string) error {
	if err := fs.MkdirAll(outputfolder, 0777); err != nil {
		return fmt.Errorf("failed to create output folder %q: %w", outputfolder, err)
	}

	if s, err := manifest.GenerateJSONSchema(); err != nil {
		return err
	} else if err := writeSchemaFile(fs, filepath.Join(outputfolder, "monaco-manifest.schema.json"), s); err != nil {
		return err
	}

	if s, err := config.GenerateJSONSchema(); err != nil {
		return err
	} else if err := writeSchemaFile(fs, filepath.Join(outputfolder, "monaco-config.schema.json"), s); err != nil {
		return err
	}

	if s, err := account.GenerateJSONSchema(); err != nil {
		return err
	} else if err := writeSchemaFile(fs, filepath.Join(outputfolder, "monaco-account-resource.schema.json"), s); err != nil {
		return err
	}

	if s, err := delete.GenerateJSONSchema(); err != nil {
		return err
	} else if err := writeSchemaFile(fs, filepath.Join(outputfolder, "monaco-delete-file.schema.json"), s); err != nil {
		return err
	}

	if s, err := accountDelete.GenerateJSONSchema(); err != nil {
		return err
	} else if err := writeSchemaFile(fs, filepath.Join(outputfolder, "monaco-account-delete-file.schema.json"), s); err != nil {
		return err
	}

	return nil
}

func writeSchemaFile(fs afero.Fs, path string, schema []byte) error {
	if err := afero.WriteFile(fs, filepath.Clean(path), schema, 0664); err != nil {
		return fmt.Errorf("failed to create schema file %q: %w", path, err)
	}

	log.Info("Generated JSON schema %q", path)
	return nil
}
