// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v2

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/slices"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

var allowedScopeParameterTypes = []string{
	refParam.ReferenceParameterType,
	valueParam.ValueParameterType,
	environment.EnvironmentVariableParameterType,
}

type LoaderContext struct {
	ProjectId       string
	Path            string
	Environments    []manifest.EnvironmentDefinition
	KnownApis       map[string]struct{}
	ParametersSerDe map[string]parameter.ParameterSerDe
}

// LoadConfigs will search a given path for configuration yamls and parses them.
// It will try to parse all configurations it finds and returns a list of parsed
// configs. If any error was encountered, the list of configs will be nil and
// only the error slice will be filled.
func LoadConfigs(fs afero.Fs, context *LoaderContext) (result []Config, errors []error) {
	filesInFolder, err := afero.ReadDir(fs, context.Path)

	if err != nil {
		return nil, []error{err}
	}

	for _, file := range filesInFolder {
		filename := file.Name()

		if file.IsDir() || !files.IsYamlFileExtension(filename) {
			continue
		}

		configs, configErrs := parseConfigs(fs, context, filepath.Join(context.Path, filename))

		if configErrs != nil {
			errors = append(errors, configErrs...)
			continue
		}

		result = append(result, configs...)

	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

type ConfigLoaderContext struct {
	*LoaderContext
	Folder string
	Path   string
}

type SingleConfigLoadContext struct {
	*ConfigLoaderContext
	Type string
}

type DefinitionParserError struct {
	Location coordinate.Coordinate
	Path     string
	Reason   string
}

func newDefinitionParserError(configId string, context *SingleConfigLoadContext, reason string) DefinitionParserError {
	return DefinitionParserError{
		Location: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Path:   context.Path,
		Reason: reason,
	}
}

type DetailedDefinitionParserError struct {
	DefinitionParserError
	EnvironmentDetails configErrors.EnvironmentDetails
}

func newDetailedDefinitionParserError(configId string, context *SingleConfigLoadContext, environment manifest.EnvironmentDefinition,
	reason string) DetailedDefinitionParserError {

	return DetailedDefinitionParserError{
		DefinitionParserError: newDefinitionParserError(configId, context, reason),
		EnvironmentDetails:    configErrors.EnvironmentDetails{Group: environment.Group, Environment: environment.Name},
	}
}

func (e DetailedDefinitionParserError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e DetailedDefinitionParserError) Environment() string {
	return e.EnvironmentDetails.Environment
}

func (e DefinitionParserError) Coordinates() coordinate.Coordinate {
	return e.Location
}

type ParameterDefinitionParserError struct {
	DetailedDefinitionParserError
	ParameterName string
}

func newParameterDefinitionParserError(name string, configId string, context *SingleConfigLoadContext,
	environment manifest.EnvironmentDefinition, reason string) ParameterDefinitionParserError {

	return ParameterDefinitionParserError{
		DetailedDefinitionParserError: newDetailedDefinitionParserError(configId, context, environment, reason),
		ParameterName:                 name,
	}
}

var (
	_ configErrors.ConfigError         = (*DefinitionParserError)(nil)
	_ configErrors.DetailedConfigError = (*DetailedDefinitionParserError)(nil)
	_ configErrors.DetailedConfigError = (*ParameterDefinitionParserError)(nil)
)

func (e ParameterDefinitionParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter definition in `%s`: %s",
		e.ParameterName, e.Path, e.Reason)
}

func (e DefinitionParserError) Error() string {
	return fmt.Sprintf("cannot parse definition in `%s`: %s",
		e.Path, e.Reason)
}

func parseConfigs(fs afero.Fs, context *LoaderContext, filePath string) (configs []Config, errors []error) {
	data, err := afero.ReadFile(fs, filePath)
	folder := filepath.Dir(filePath)

	if err != nil {
		return nil, []error{err}
	}

	definition := topLevelDefinition{}

	err = yaml.UnmarshalStrict(data, &definition)

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("field config not found in type %s", getTopLevelDefinitionYamlTypeName())) {
			return nil, []error{
				fmt.Errorf("config '%s' is not valid v2 configuration - you may be loading v1 configs, please 'convert' to v2:\n%w", filePath, err),
			}
		}

		return nil, []error{
			fmt.Errorf("failed to load config '%s':\n%w", filePath, err),
		}
	}

	if len(definition.Configs) == 0 {
		return nil, []error{
			fmt.Errorf("no configurations found in file '%s'", filePath),
		}
	}

	configLoaderContext := &ConfigLoaderContext{
		LoaderContext: context,
		Folder:        folder,
		Path:          filePath,
	}

	for _, config := range definition.Configs {
		result, definitionErrors := parseDefinition(fs, configLoaderContext, config.Id, config)

		if definitionErrors != nil {
			errors = append(errors, definitionErrors...)
			continue
		}

		configs = append(configs, result...)
	}

	if errors != nil {
		return nil, errors
	}

	return configs, nil
}

// parseDefinition parses a single config entry
func parseDefinition(
	fs afero.Fs,
	context *ConfigLoaderContext,
	configId string,
	definition topLevelConfigDefinition,
) ([]Config, []error) {

	results := make([]Config, 0)
	var errors []error

	if b, e := definition.Type.isSound(context.KnownApis); !b {
		return nil, append(errors, e)
	}

	singleConfigContext := &SingleConfigLoadContext{
		ConfigLoaderContext: context,
		Type:                definition.Type.GetApiType(),
	}

	groupOverrideMap := toGroupOverrideMap(definition.GroupOverrides)
	environmentOverrideMap := toEnvironmentOverrideMap(definition.EnvironmentOverrides)

	for _, environment := range context.Environments {
		result, definitionErrors := parseDefinitionForEnvironment(fs, singleConfigContext, configId, environment,
			definition, groupOverrideMap, environmentOverrideMap)

		if definitionErrors != nil {
			errors = append(errors, definitionErrors...)
			continue
		}

		results = append(results, result)
	}

	if errors != nil {
		return nil, errors
	}

	return results, nil
}

func toEnvironmentOverrideMap(environments []environmentOverride) map[string]environmentOverride {
	result := make(map[string]environmentOverride)

	for _, env := range environments {
		result[env.Environment] = env
	}

	return result
}

func toGroupOverrideMap(groups []groupOverride) map[string]groupOverride {
	result := make(map[string]groupOverride)

	for _, group := range groups {
		result[group.Group] = group
	}

	return result
}

func parseDefinitionForEnvironment(
	fs afero.Fs,
	context *SingleConfigLoadContext,
	configId string,
	environment manifest.EnvironmentDefinition,
	definition topLevelConfigDefinition,
	groupOverrides map[string]groupOverride,
	environmentOverride map[string]environmentOverride,
) (Config, []error) {

	configDefinition := configDefinition{
		Parameters: make(map[string]configParameter),
	}

	applyOverrides(&configDefinition, definition.Config)

	if override, found := groupOverrides[environment.Group]; found {
		applyOverrides(&configDefinition, override.Override)
	}

	if override, found := environmentOverride[environment.Name]; found {
		applyOverrides(&configDefinition, override.Override)
	}

	configDefinition.Template = filepath.FromSlash(configDefinition.Template)

	return getConfigFromDefinition(fs, context, configId, environment, configDefinition, definition.Type)
}

func applyOverrides(base *configDefinition, override configDefinition) {
	if override.Name != nil {
		base.Name = override.Name
	}

	if override.Template != "" {
		base.Template = override.Template
	}

	if override.Skip != nil {
		base.Skip = override.Skip
	}

	for name, param := range override.Parameters {
		base.Parameters[name] = param
	}

}

func getConfigFromDefinition(
	fs afero.Fs,
	context *SingleConfigLoadContext,
	configId string,
	environment manifest.EnvironmentDefinition,
	definition configDefinition,
	configType typeDefinition,
) (Config, []error) {

	if definition.Template == "" {
		return Config{}, []error{
			newDetailedDefinitionParserError(configId, context, environment, "missing property `template`"),
		}
	}

	template, err := template.LoadTemplate(fs, filepath.Join(context.Folder, definition.Template))

	var errors []error

	if err != nil {
		errors = append(errors, newDetailedDefinitionParserError(configId, context, environment, fmt.Sprintf("error while loading template: `%s`", err)))
	}

	parameters, parameterErrors := parseParametersAndReferences(context, environment, configId,
		definition.Parameters)

	if parameterErrors != nil {
		errors = append(errors, parameterErrors...)
		parameters = make(map[string]parameter.Parameter)
	}

	skipConfig := false

	if definition.Skip != nil {
		skip, err := parseSkip(context, environment, configId, definition.Skip)
		if err == nil {
			skipConfig = skip
		} else {
			errors = append(errors, err)
		}
	}

	if definition.Name != nil {
		name, err := parseParameter(context, environment, configId, NameParameter, definition.Name)
		if err != nil {
			errors = append(errors, err)
		} else {
			parameters[NameParameter] = name
		}

	} else {
		errors = append(errors, newDetailedDefinitionParserError(configId, context, environment, "missing parameter `name`"))
	}

	if errors != nil {
		return Config{}, errors
	}

	if configType.isSettings() {
		scopeParam, err := parseParameter(context, environment, configId, ScopeParameter, configType.Settings.Scope)
		if err != nil {
			return Config{}, []error{fmt.Errorf("failed to parse scope: %w", err)}
		}

		if !slices.Contains(allowedScopeParameterTypes, scopeParam.GetType()) {
			return Config{}, []error{fmt.Errorf("failed to parse scope: Cannot use parameter-type '%s' within the scope. Allowed types: %v", scopeParam.GetType(), allowedScopeParameterTypes)}
		}

		parameters[ScopeParameter] = scopeParam
	}

	return Config{
		Template: template,
		Coordinate: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Type: Type{
			SchemaId:      configType.Settings.Schema,
			SchemaVersion: configType.Settings.SchemaVersion,
			Api:           configType.Api,
		},
		Group:       environment.Group,
		Environment: environment.Name,
		Parameters:  parameters,
		Skip:        skipConfig,
	}, nil
}

func parseSkip(
	context *SingleConfigLoadContext,
	environmentDefinition manifest.EnvironmentDefinition,
	configId string,
	param interface{},
) (bool, error) {
	parsed, err := parseParameter(context, environmentDefinition, configId, SkipParameter, param)
	if err != nil {
		return false, err
	}

	if !isSupportedParamTypeForSkip(parsed) {
		return false, newParameterDefinitionParserError(SkipParameter, configId, context, environmentDefinition, "must be of type 'value' or 'environment'")
	}

	resolved, err := parsed.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Group:         environmentDefinition.Group,
		Environment:   environmentDefinition.Name,
		ParameterName: SkipParameter,
	})
	if err != nil {
		return false, newParameterDefinitionParserError(SkipParameter, configId, context, environmentDefinition, fmt.Sprintf("failed to resolve value: %s", err))
	}

	retVal, err := strconv.ParseBool(fmt.Sprintf("%v", resolved))
	if err != nil {
		return false, newParameterDefinitionParserError(SkipParameter, configId, context, environmentDefinition, fmt.Sprintf("resolved value can only be 'true' or 'false' (current value is: '%v'", resolved))
	}

	return retVal, err
}

// isSupportedParamTypeForSkip check is 'skip' section of configuration supports specified param type
func isSupportedParamTypeForSkip(p parameter.Parameter) bool {
	switch p.GetType() {
	case valueParam.ValueParameterType:
		return true
	case environment.EnvironmentVariableParameterType:
		return true
	default:
		return false
	}
}

// References holds coordinate-string -> coordinate
type References map[string]coordinate.Coordinate

func parseParametersAndReferences(context *SingleConfigLoadContext, environment manifest.EnvironmentDefinition,
	configId string, parameterMap map[string]configParameter) (Parameters, []error) {

	parameters := make(map[string]parameter.Parameter)
	var errors []error

	for name, param := range parameterMap {
		if _, found := parameters[name]; found {
			errors = append(errors, newDefinitionParserError(configId, context, fmt.Sprintf("duplicate parameter `%s`", name)))
			continue
		}

		err := validateParameterName(context, environment, configId, name)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		result, err := parseParameter(context, environment, configId, name, param)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		parameters[name] = result
	}

	if errors != nil {
		return nil, errors
	}

	return parameters, nil
}

func validateParameterName(context *SingleConfigLoadContext, environment manifest.EnvironmentDefinition, configId string, name string) error {

	for _, parameterName := range ReservedParameterNames {
		if name == parameterName {
			return newParameterDefinitionParserError(name, configId, context, environment,
				fmt.Sprintf("parameter name `%s` is not allowed (reserved)", parameterName))
		}
	}

	return nil
}

func parseParameter(context *SingleConfigLoadContext, environment manifest.EnvironmentDefinition,
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
			Coordinate: coordinate.Coordinate{
				Project:  context.ProjectId,
				Type:     context.Type,
				ConfigId: configId,
			},
			ParameterName: name,
			Value:         maps.ToStringMap(val),
		})
	}

	return valueParam.New(param), nil
}

// TODO come up with better way to handle this, as this is a hack
func arrayToReferenceParameter(context *SingleConfigLoadContext, environment manifest.EnvironmentDefinition,
	configId string, parameterName string, arr []interface{}) (parameter.Parameter, error) {
	if len(arr) == 0 || len(arr) > 4 {
		return nil, newParameterDefinitionParserError(parameterName, configId, context, environment,
			fmt.Sprintf("short references must have between 1 and 4 elements. you provided `%d`", len(arr)))
	}

	project := context.ProjectId
	configType := context.Type
	config := configId
	var property string

	switch len(arr) {
	case 1:
		property = toString(arr[0])
	case 2:
		config = toString(arr[0])
		property = toString(arr[1])
	case 3:
		configType = toString(arr[0])
		config = toString(arr[1])
		property = toString(arr[2])
	case 4:
		project = toString(arr[0])
		configType = toString(arr[1])
		config = toString(arr[2])
		property = toString(arr[3])
	}

	return refParam.New(project, configType, config, property), nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
