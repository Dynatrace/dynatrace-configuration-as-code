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

package loader

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/attribute"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

type LoaderContext struct {
	ProjectId       string
	Path            string
	Environments    manifest.Environments
	KnownApis       map[string]struct{}
	ParametersSerDe map[string]parameter.ParameterSerDe
}

// configFileLoaderContext is a context for each config-file
type configFileLoaderContext struct {
	*LoaderContext
	Folder string
	Path   string
}

// singleConfigEntryLoadContext is a context for each config-entry within a config-file
type singleConfigEntryLoadContext struct {
	*configFileLoaderContext
	Type string
}

// LoadConfigFile loads a single configuration file and returns all configs defined in that file.
// The returned configs contain all variants for project/environment overwrites passed in the [LoaderContext]
func LoadConfigFile(ctx context.Context, fs afero.Fs, context *LoaderContext, filePath string) ([]config.Config, []error) {
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, []error{newLoadError(filePath, err)}
	}

	// validate that the config does not contain the key 'config'. This key is used in monaco v1 and could indicate
	// that the user tries to deploy monaco v1 configuration using monaco v2.
	var content map[string]any
	if err := yaml.Unmarshal(data, &content); err != nil {
		return nil, []error{newLoadError(filePath, err)}
	}
	if content["config"] != nil {
		return nil, []error{
			newLoadError(filePath, fmt.Errorf("config is not a valid v2 configuration - you may be loading v1 configs, please 'convert' to v2: %w", err)),
		}
	}

	// Validate that the config has only accounts OR configs specified. We do not allow defining both in one file.
	if loader.HasAnyAccountKeyDefined(content) {
		if content["configs"] != nil {
			return nil, []error{newLoadError(filePath, ErrMixingConfigs)}
		}

		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateWarn, nil, fmt.Sprintf("File %q appears to be an account resource file, skipping loading", filePath), nil)
		log.With(attribute.Any("file", filePath)).WarnContext(ctx, "File %q appears to be an account resource file, skipping loading", filePath)
		return []config.Config{}, nil
	}

	// Actually load the configs
	loadedConfigEntries, err := loadConfigDefinitions(data)
	if err != nil {
		return nil, []error{newLoadError(filePath, err)}
	}

	configLoaderContext := &configFileLoaderContext{
		LoaderContext: context,
		Folder:        filepath.Dir(filePath),
		Path:          filePath,
	}

	var errs []error
	var configs []config.Config

	for _, cgf := range loadedConfigEntries {

		result, definitionErrors := parseConfigEntry(fs, configLoaderContext, cgf.Id, cgf)

		configs = append(configs, result...)

		if len(definitionErrors) > 0 {
			errs = append(errs, definitionErrors...)
		}
	}

	if errs != nil {
		return configs, errs
	}

	return configs, nil
}

func loadConfigDefinitions(data []byte) ([]persistence.TopLevelConfigDefinition, error) {

	definition := persistence.TopLevelDefinition{}
	err := yaml.UnmarshalStrict(data, &definition)

	if err != nil {
		return nil, err
	}

	if len(definition.Configs) == 0 {
		return nil, fmt.Errorf("no configurations found in file")
	}

	return definition.Configs, nil
}
