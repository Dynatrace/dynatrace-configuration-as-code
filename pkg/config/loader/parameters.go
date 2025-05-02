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
	"fmt"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

var allowedScopeParameterTypes = []string{
	refParam.ReferenceParameterType,
	valueParam.ValueParameterType,
	envParam.EnvironmentVariableParameterType,
}

// isSupportedParamTypeForSkip check is 'skip' section of configuration supports specified param type
func isSupportedParamTypeForSkip(p parameter.Parameter) bool {
	switch p.GetType() {
	case valueParam.ValueParameterType:
		return true
	case envParam.EnvironmentVariableParameterType:
		return true
	default:
		return false
	}
}

// References holds coordinate-string -> coordinate
type References map[string]coordinate.Coordinate

func parseParametersAndReferences(fs afero.Fs, context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
	configId string, parameterMap map[string]persistence.ConfigParameter) (config.Parameters, []error) {

	parameters := make(map[string]parameter.Parameter)
	var errs []error

	for name, param := range parameterMap {
		if _, found := parameters[name]; found {
			errs = append(errs, newDefinitionParserError(configId, context, fmt.Sprintf("duplicate parameter `%s`", name)))
			continue
		}

		err := validateParameterName(context, environment, configId, name)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		result, err := parseParameter(fs, context, environment, configId, name, param)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		err = validateParameter(context, name, result)
		if err != nil {
			errs = append(errs, newDetailedDefinitionParserError(configId, context, environment, err.Error()))
			continue
		}

		parameters[name] = result
	}

	if errs != nil {
		return nil, errs
	}

	return parameters, nil
}

func validateParameterName(context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition, configId string, name string) error {

	for _, parameterName := range config.ReservedParameterNames {
		if name == parameterName {
			return newParameterDefinitionParserError(name, configId, context, environment,
				fmt.Sprintf("parameter name `%s` is not allowed (reserved)", parameterName))
		}
	}

	return nil
}

func parseParameter(fs afero.Fs, context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
	configId string, name string, param interface{}) (parameter.Parameter, error) {

	if val, ok := param.([]interface{}); ok {
		ref, err := arrayToReferenceParameter(context, environment, configId, name, val)

		if err != nil {
			return nil, err
		}

		return ref, nil
	} else if val, ok := param.(map[interface{}]interface{}); ok {
		parameterType := toString(val["type"])
		serDe, found := context.ParametersSerDe[parameterType]

		if !found {
			return nil, newParameterDefinitionParserError(name, configId, context, environment,
				fmt.Sprintf("unknown parameter type `%s`", parameterType))
		}

		return serDe.Deserializer(parameter.ParameterParserContext{
			WorkingDirectory: context.Folder,
			Coordinate: coordinate.Coordinate{
				Project:  context.ProjectId,
				Type:     context.Type,
				ConfigId: configId,
			},
			Fs:            fs,
			ParameterName: name,
			Value:         maps.ToStringMap(val),
		})
	}

	return valueParam.New(param), nil
}

// TODO come up with better way to handle this, as this is a hack
func arrayToReferenceParameter(context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
	configId string, parameterName string, arr []interface{}) (parameter.Parameter, error) {
	if len(arr) == 0 || len(arr) > 4 {
		return nil, newParameterDefinitionParserError(parameterName, configId, context, environment,
			fmt.Sprintf("short references must have between 1 and 4 elements. you provided `%d`", len(arr)))
	}

	project := context.ProjectId
	configType := context.Type
	cfg := configId
	var property string

	switch len(arr) {
	case 1:
		property = toString(arr[0])
	case 2:
		cfg = toString(arr[0])
		property = toString(arr[1])
	case 3:
		configType = toString(arr[0])
		cfg = toString(arr[1])
		property = toString(arr[2])
	case 4:
		project = toString(arr[0])
		configType = toString(arr[1])
		cfg = toString(arr[2])
		property = toString(arr[3])
	}

	return refParam.New(project, configType, cfg, property), nil
}

func validateParameter(ctx *singleConfigEntryLoadContext, paramName string, param parameter.Parameter) error {
	if _, isAPI := ctx.KnownApis[ctx.Type]; isAPI {
		for _, ref := range param.GetReferences() {
			if _, referencesAPI := ctx.KnownApis[ref.Config.Type]; !referencesAPI &&
				ref.Property == config.IdParameter &&
				(ref.Config.Type != "builtin:management-zones" || !featureflags.ManagementZoneSettingsNumericIDs.Enabled()) { // leniently handle Management Zone numeric IDs which are the same for Settings
				return fmt.Errorf("config api type (%s) configuration can only reference IDs of other config api types - parameter %q references %q type", ctx.Type, paramName, ref.Config.Type)
			}
		}
	}
	return nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
