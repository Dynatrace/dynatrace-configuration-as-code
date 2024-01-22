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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/internal/persistence"
	"github.com/spf13/afero"
	"path/filepath"
	"slices"
	"strconv"
)

// parseConfigEntry parses a single config entry
func parseConfigEntry(
	fs afero.Fs,
	context *configFileLoaderContext,
	configId string,
	definition persistence.TopLevelConfigDefinition,
) ([]config.Config, []error) {

	typ, err := definition.Type.GetApiType()
	if err != nil {
		return nil, []error{fmt.Errorf("failed to parse config entry %q: %w", configId, err)}
	}

	singleConfigContext := &singleConfigEntryLoadContext{
		configFileLoaderContext: context,
		Type:                    typ,
	}

	if e := definition.Type.IsSound(context.KnownApis); e != nil {
		return nil, []error{newDefinitionParserError(configId, singleConfigContext, e.Error())}
	}

	groupOverrideMap := toGroupOverrideMap(definition.GroupOverrides)
	environmentOverrideMap := toEnvironmentOverrideMap(definition.EnvironmentOverrides)

	var results []config.Config
	var errs []error
	for _, env := range context.Environments {

		result, definitionErrors := parseDefinitionForEnvironment(fs, singleConfigContext, configId, env, definition, groupOverrideMap, environmentOverrideMap)

		if definitionErrors != nil {
			errs = append(errs, definitionErrors...)
			continue
		}

		results = append(results, result)
	}

	if len(errs) != 0 {
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

	tmpl, err := template.NewFileTemplate(fs, filepath.Join(context.Folder, definition.Template))

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
			return config.Config{}, []error{fmt.Errorf("failed to parse scope: cannot use parameter-type %q within the scope. Allowed types: %v", scopeParam.GetType(), allowedScopeParameterTypes)}
		}

		parameters[config.ScopeParameter] = scopeParam
	}

	if configType.IsClassic() {
		a, err := persistence.UnmarshalApiType(configType.Api)
		if err != nil {
			return config.Config{}, []error{fmt.Errorf("failed to parse config: %w", err)}
		}

		if a.Scope != nil {
			scopeParam, err := parseParameter(context, environment, configId, config.ScopeParameter, a.Scope)
			if err != nil {
				return config.Config{}, []error{fmt.Errorf("failed to parse scope: %w", err)}
			}

			if !slices.Contains(allowedScopeParameterTypes, scopeParam.GetType()) {
				return config.Config{}, []error{fmt.Errorf("failed to parse api: cannot use parameter-type %q within the scope. Allowed types: %v", scopeParam.GetType(), allowedScopeParameterTypes)}
			}

			parameters[config.ScopeParameter] = scopeParam
		}
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
		a, err := persistence.UnmarshalApiType(typeDef.Api)
		if err != nil {
			return nil, err
		}

		return config.ClassicApiType{
			Api: a.Name,
		}, nil

	case typeDef.IsAutomation():
		return config.AutomationType{
			Resource: typeDef.Automation.Resource,
		}, nil
	case typeDef.IsBucket():
		return config.BucketType{}, nil

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
