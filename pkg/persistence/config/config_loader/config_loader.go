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

package config_loader

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/internal/config_persistence"
	"path/filepath"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type LoaderContext struct {
	ProjectId       string
	Path            string
	Environments    []manifest.EnvironmentDefinition
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

// LoadConfig loads a single configuration file
// The configuration file might contain multiple config entries
func LoadConfig(fs afero.Fs, context *LoaderContext, filePath string) ([]config.Config, []error) {
	definedConfigEntries, err := parseFile(fs, filePath)
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

	for _, cgf := range definedConfigEntries {

		result, definitionErrors := parseConfigEntry(fs, configLoaderContext, cgf.Id, cgf)

		if len(definitionErrors) > 0 {
			errs = append(errs, definitionErrors...)
			continue
		}

		configs = append(configs, result...)
	}

	if errs != nil {
		return nil, errs
	}

	return configs, nil
}

func parseFile(fs afero.Fs, filePath string) ([]config_persistence.TopLevelConfigDefinition, error) {
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, err
	}

	definition := config_persistence.TopLevelDefinition{}
	err = yaml.UnmarshalStrict(data, &definition)

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("field config not found in type %s", config_persistence.GetTopLevelDefinitionYamlTypeName())) {
			return nil, fmt.Errorf("config '%s' is not valid v2 configuration - you may be loading v1 configs, please 'convert' to v2:\n%w", filePath, err)
		}

		return nil, err
	}

	if len(definition.Configs) == 0 {
		return nil, fmt.Errorf("no configurations found in file '%s'", filePath)
	}

	return definition.Configs, nil
}
