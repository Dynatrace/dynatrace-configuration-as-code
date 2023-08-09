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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/slices"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/internal/persistence"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
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
	data, err := afero.ReadFile(fs, filePath)
	if err != nil {
		return nil, []error{newLoadError(context.Path, err)}
	}

	definition := persistence.TopLevelDefinition{}
	err = yaml.UnmarshalStrict(data, &definition)

	if err != nil {
		if strings.Contains(err.Error(), fmt.Sprintf("field config not found in type %s", persistence.GetTopLevelDefinitionYamlTypeName())) {
			return nil, []error{
				newLoadError(
					context.Path,
					fmt.Errorf("config '%s' is not valid v2 configuration - you may be loading v1 configs, please 'convert' to v2:\n%w", filePath, err),
				),
			}
		}

		return nil, []error{newLoadError(filePath, err)}
	}

	if len(definition.Configs) == 0 {
		return nil, []error{newLoadError(filePath, fmt.Errorf("no configurations found in file '%s'", filePath))}
	}

	configLoaderContext := &configFileLoaderContext{
		LoaderContext: context,
		Folder:        filepath.Dir(filePath),
		Path:          filePath,
	}

	var errs []error
	var configs []config.Config

	for _, cnf := range definition.Configs {

		result, definitionErrors := parseDefinition(fs, configLoaderContext, cnf.Id, cnf)

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

// parseDefinition parses a single config entry
func parseDefinition(
	fs afero.Fs,
	context *configFileLoaderContext,
	configId string,
	definition persistence.TopLevelConfigDefinition,
) ([]config.Config, []error) {

	results := make([]config.Config, 0)
	var errs []error

	singleConfigContext := &singleConfigEntryLoadContext{
		configFileLoaderContext: context,
		Type:                    definition.Type.GetApiType(),
	}

	if e := definition.Type.IsSound(context.KnownApis); e != nil {
		return nil, append(errs, newDefinitionParserError(configId, singleConfigContext, e.Error()))
	}

	groupOverrideMap := toGroupOverrideMap(definition.GroupOverrides)
	environmentOverrideMap := toEnvironmentOverrideMap(definition.EnvironmentOverrides)

	for _, env := range context.Environments {
		result, definitionErrors := parseDefinitionForEnvironment(fs, singleConfigContext, configId, env,
			definition, groupOverrideMap, environmentOverrideMap)

		if definitionErrors != nil {
			errs = append(errs, definitionErrors...)
			continue
		}

		results = append(results, result)
	}

	if errs != nil {
		return nil, errs
	}

	return results, nil
}

func toEnvironmentOverrideMap(environments []persistence.EnvironmentOverride) map[string]persistence.EnvironmentOverride {
	result := make(map[string]persistence.EnvironmentOverride)

	for _, env := range environments {
		result[env.Environment] = env
	}

	return result
}

func toGroupOverrideMap(groups []persistence.GroupOverride) map[string]persistence.GroupOverride {
	result := make(map[string]persistence.GroupOverride)

	for _, group := range groups {
		result[group.Group] = group
	}

	return result
}

func parseDefinitionForEnvironment(
	fs afero.Fs,
	context *singleConfigEntryLoadContext,
	configId string,
	environment manifest.EnvironmentDefinition,
	definition persistence.TopLevelConfigDefinition,
	groupOverrides map[string]persistence.GroupOverride,
	environmentOverride map[string]persistence.EnvironmentOverride,
) (config.Config, []error) {

	configDefinition := persistence.ConfigDefinition{
		Parameters:     make(map[string]persistence.ConfigParameter),
		OriginObjectId: definition.Config.OriginObjectId,
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

func applyOverrides(base *persistence.ConfigDefinition, override persistence.ConfigDefinition) {
	if override.Name != nil {
		base.Name = override.Name
	}

	if override.Template != "" {
		base.Template = override.Template
	}

	if override.Skip != nil {
		base.Skip = override.Skip
	}

	if override.OriginObjectId != "" {
		base.OriginObjectId = override.OriginObjectId
	}

	for name, param := range override.Parameters {
		base.Parameters[name] = param
	}

}

func getConfigFromDefinition(
	fs afero.Fs,
	context *singleConfigEntryLoadContext,
	configId string,
	environment manifest.EnvironmentDefinition,
	definition persistence.ConfigDefinition,
	configType persistence.TypeDefinition,
) (config.Config, []error) {

	if definition.Template == "" {
		return config.Config{}, []error{
			newDetailedDefinitionParserError(configId, context, environment, "missing property `template`"),
		}
	}

	tmpl, err := template.LoadTemplate(fs, filepath.Join(context.Folder, definition.Template))

	var errs []error

	if err != nil {
		errs = append(errs, newDetailedDefinitionParserError(configId, context, environment, fmt.Sprintf("error while loading template: `%s`", err)))
	}

	parameters, parameterErrors := parseParametersAndReferences(context, environment, configId,
		definition.Parameters)

	if parameterErrors != nil {
		errs = append(errs, parameterErrors...)
		parameters = make(map[string]parameter.Parameter)
	}

	skipConfig := false

	if definition.Skip != nil {
		skip, err := parseSkip(context, environment, configId, definition.Skip)
		if err == nil {
			skipConfig = skip
		} else {
			errs = append(errs, err)
		}
	}

	t, err := getType(configType)
	if err != nil {
		return config.Config{}, []error{fmt.Errorf("failed to parse type of config %q: %w", configId, err)}
	}

	if definition.Name != nil {
		name, err := parseParameter(context, environment, configId, config.NameParameter, definition.Name)
		if err != nil {
			errs = append(errs, err)
		} else {
			parameters[config.NameParameter] = name
		}

	} else if t.ID() == config.ClassicApiTypeId {
		errs = append(errs, newDetailedDefinitionParserError(configId, context, environment, "missing parameter `name`"))
	}

	if errs != nil {
		return config.Config{}, errs
	}

	if configType.IsSettings() {
		scopeParam, err := parseParameter(context, environment, configId, config.ScopeParameter, configType.Settings.Scope)
		if err != nil {
			return config.Config{}, []error{fmt.Errorf("failed to parse scope: %w", err)}
		}

		if !slices.Contains(allowedScopeParameterTypes, scopeParam.GetType()) {
			return config.Config{}, []error{fmt.Errorf("failed to parse scope: Cannot use parameter-type '%s' within the scope. Allowed types: %v", scopeParam.GetType(), allowedScopeParameterTypes)}
		}

		parameters[config.ScopeParameter] = scopeParam
	}

	return config.Config{
		Template: tmpl,
		Coordinate: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Type:           t,
		Group:          environment.Group,
		Environment:    environment.Name,
		Parameters:     parameters,
		Skip:           skipConfig,
		OriginObjectId: definition.OriginObjectId,
	}, nil
}

func getType(typeDef persistence.TypeDefinition) (config.Type, error) {
	switch {
	case typeDef.IsSettings():
		return config.SettingsType{
			SchemaId:      typeDef.Settings.Schema,
			SchemaVersion: typeDef.Settings.SchemaVersion,
		}, nil

	case typeDef.IsClassic():

		if typeDef.Api == persistence.ApiTypeBucket {
			return config.BucketType{}, nil
		}

		return config.ClassicApiType{
			Api: typeDef.Api,
		}, nil

	case typeDef.IsEntities():
		return config.EntityType{
			EntitiesType: typeDef.Entities.EntitiesType,
		}, nil
	case typeDef.IsAutomation():
		return config.AutomationType{
			Resource: typeDef.Automation.Resource,
		}, nil

	default:
		return nil, errors.New("unknown type")
	}
}

func parseSkip(
	context *singleConfigEntryLoadContext,
	environmentDefinition manifest.EnvironmentDefinition,
	configId string,
	param interface{},
) (bool, error) {
	parsed, err := parseParameter(context, environmentDefinition, configId, config.SkipParameter, param)
	if err != nil {
		return false, err
	}

	if !isSupportedParamTypeForSkip(parsed) {
		return false, newParameterDefinitionParserError(config.SkipParameter, configId, context, environmentDefinition, "must be of type 'value' or 'environment'")
	}

	resolved, err := parsed.ResolveValue(parameter.ResolveContext{
		ConfigCoordinate: coordinate.Coordinate{
			Project:  context.ProjectId,
			Type:     context.Type,
			ConfigId: configId,
		},
		Group:         environmentDefinition.Group,
		Environment:   environmentDefinition.Name,
		ParameterName: config.SkipParameter,
	})
	if err != nil {
		return false, newParameterDefinitionParserError(config.SkipParameter, configId, context, environmentDefinition, fmt.Sprintf("failed to resolve value: %s", err))
	}

	retVal, err := strconv.ParseBool(fmt.Sprintf("%v", resolved))
	if err != nil {
		return false, newParameterDefinitionParserError(config.SkipParameter, configId, context, environmentDefinition, fmt.Sprintf("resolved value can only be 'true' or 'false' (current value is: '%v'", resolved))
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

func parseParametersAndReferences(context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
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

		result, err := parseParameter(context, environment, configId, name, param)
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

func parseParameter(context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
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
func arrayToReferenceParameter(context *singleConfigEntryLoadContext, environment manifest.EnvironmentDefinition,
	configId string, parameterName string, arr []interface{}) (parameter.Parameter, error) {
	if len(arr) == 0 || len(arr) > 4 {
		return nil, newParameterDefinitionParserError(parameterName, configId, context, environment,
			fmt.Sprintf("short references must have between 1 and 4 elements. you provided `%d`", len(arr)))
	}

	project := context.ProjectId
	configType := context.Type
	cnf := configId
	var property string

	switch len(arr) {
	case 1:
		property = toString(arr[0])
	case 2:
		cnf = toString(arr[0])
		property = toString(arr[1])
	case 3:
		configType = toString(arr[0])
		cnf = toString(arr[1])
		property = toString(arr[2])
	case 4:
		project = toString(arr[0])
		configType = toString(arr[1])
		cnf = toString(arr[2])
		property = toString(arr[3])
	}

	return refParam.New(project, configType, cnf, property), nil
}

func validateParameter(ctx *singleConfigEntryLoadContext, paramName string, param parameter.Parameter) error {
	if _, isAPI := ctx.KnownApis[ctx.Type]; isAPI {
		for _, ref := range param.GetReferences() {
			if _, referencesAPI := ctx.KnownApis[ref.Config.Type]; !referencesAPI &&
				ref.Property == config.IdParameter &&
				!(ref.Config.Type == "builtin:management-zones" && featureflags.ManagementZoneSettingsNumericIDs().Enabled()) { // leniently handle Management Zone numeric IDs which are the same for Settings
				return fmt.Errorf("config api type (%s) configuration can only reference IDs of other config api types - parameter %q references %q type", ctx.Type, paramName, ref.Config.Type)
			}
		}
	}
	return nil
}

func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

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
