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
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

var ErrMixingConfigs = errors.New("mixing both configurations and account resources is not allowed")

func newLoadError(path string, err error) configErrors.ConfigLoaderError {
	return configErrors.ConfigLoaderError{
		Path: path,
		Err:  err,
	}
}

func newDefinitionParserError(configId string, context *singleConfigEntryLoadContext, reason string) configErrors.DefinitionParserError {
	return configErrors.DefinitionParserError{
		Location: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Path:   context.Path,
		Reason: reason,
	}
}

func newDetailedDefinitionParserError(configId string, context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
	reason string) configErrors.DetailedDefinitionParserError {

	return configErrors.DetailedDefinitionParserError{
		DefinitionParserError: newDefinitionParserError(configId, context, reason),
		EnvironmentDetails:    configErrors.EnvironmentDetails{Group: environment.Group, Environment: environment.Name},
	}
}

func newParameterDefinitionParserError(name string, configId string, context *singleConfigEntryLoadContext,
	environment manifest.EnvironmentDefinition, reason string) configErrors.ParameterDefinitionParserError {

	return configErrors.ParameterDefinitionParserError{
		DetailedDefinitionParserError: newDetailedDefinitionParserError(configId, context, environment, reason),
		ParameterName:                 name,
	}
}
