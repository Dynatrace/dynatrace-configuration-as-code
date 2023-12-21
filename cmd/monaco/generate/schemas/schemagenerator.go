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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/mutlierror"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/topologysort"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/converter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	deploy "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestLoader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	manifestWriter "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/writer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	sortErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/spf13/afero"
	"path/filepath"
)

var errorStructs = []interface{}{
	json.JsonValidationError{},
	configErrors.InvalidJsonError{},
	configErrors.ConfigLoaderError{},
	configErrors.DefinitionParserError{},
	configErrors.DetailedDefinitionParserError{},
	configErrors.ParameterDefinitionParserError{},
	configErrors.ConfigWriterError{},
	configErrors.DetailedConfigWriterError{},
	parameter.ParameterParserError{},
	parameter.ParameterWriterError{},
	parameter.ParameterResolveValueError{},
	reference.UnresolvedReferenceError{},
	converter.ConvertConfigError{},
	converter.ReferenceParseError{},
	converter.TemplateConversionError{},
	delete.DeleteEntryParserError{},
	deploy.EnvironmentDeploymentErrors{},
	deploy.DeploymentErrors{},
	deploy.ConfigDeployErr{},
	manifestLoader.ManifestLoaderError{},
	manifestLoader.EnvironmentLoaderError{},
	manifestLoader.ProjectLoaderError{},
	manifestWriter.ManifestWriterError{},
	project.DuplicateConfigIdentifierError{},
	sortErrors.CircualDependencyProjectSortError{},
	sortErrors.CircularDependencyConfigSortError{},
	graph.CyclicDependencyError{},
	topologysort.TopologySortError{},
	mutlierror.MultiError{},
	rest.RespError{},
}

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

	return generateErrorSchemaFiles(fs, outputfolder)
}

func generateErrorSchemaFiles(fs afero.Fs, outputfolder string) error {
	errorsFolder := filepath.Join(outputfolder, "errors")
	if err := fs.MkdirAll(errorsFolder, 0777); err != nil {
		return fmt.Errorf("failed to generate Error type schemas: %w", err)
	}
	for _, v := range errorStructs {
		if s, err := json.GenerateJSONSchemaString(v); err != nil {
			return fmt.Errorf("failed to generate schema for error type %T: %w", v, err)
		} else {
			filename := fmt.Sprintf("monaco-error.%T.schema.json", v)
			if err := writeSchemaFile(fs, filepath.Join(errorsFolder, filename), s); err != nil {
				return err
			}
		}
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
