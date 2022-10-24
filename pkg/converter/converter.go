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

package converter

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	listParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/list"
	projectV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v1"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"regexp"
	"strings"

	configV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	configV2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	envParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	environmentV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectV2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

const (
	// default group is used, when a legacy config does not specify a group. All
	// new configs are required to be in a group.
	DefaultGroup = "default"
)

type ConverterContext struct {
	Fs afero.Fs
}

type ConfigConvertContext struct {
	*ConverterContext
	ProjectId             string
	KnownListParameterIds map[string]struct{}
	V1Apis                api.ApiMap
}

type ProjectConverterError struct {
	Project string
	Reason  string
}

func (e ProjectConverterError) Error() string {
	return fmt.Sprintf("%s: cannot convert project: %s", e.Project, e.Reason)
}

type ConvertConfigError struct {
	Location coordinate.Coordinate
	Reason   string
}

func newConvertConfigError(coord coordinate.Coordinate, reason string) ConvertConfigError {
	return ConvertConfigError{
		Location: coord,
		Reason:   reason,
	}
}

func (e ConvertConfigError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e ConvertConfigError) Error() string {
	return fmt.Sprintf("cannot convert config: %s", e.Reason)
}

type ReferenceParserError struct {
	Location      coordinate.Coordinate
	ParameterName string
	Reason        string
}

func newReferenceParserError(projectId string, config configV1.Config, parameterName string, reason string) ReferenceParserError {
	return ReferenceParserError{
		Location: coordinate.Coordinate{
			Project: projectId,
			Api:     config.GetApi().GetId(),
			Config:  config.GetId(),
		},
		ParameterName: parameterName,
		Reason:        reason,
	}
}

func (e ReferenceParserError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e ReferenceParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse reference: %s",
		e.ParameterName, e.Reason)
}

var (
	_ configErrors.ConfigError = (*ConvertConfigError)(nil)
	_ configErrors.ConfigError = (*ReferenceParserError)(nil)
)

func Convert(context ConverterContext, environments map[string]environmentV1.Environment,
	projects []projectV1.Project) (manifest.Manifest, []projectV2.Project, []error) {
	environmentDefinitions := convertEnvironments(environments)
	projectDefinitions, convertedProjects, errors := convertProjects(&context, environmentDefinitions, projects)

	return manifest.Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
	}, convertedProjects, errors
}

func convertProjects(context *ConverterContext, environments map[string]manifest.EnvironmentDefinition,
	projects []projectV1.Project) (manifest.ProjectDefinitionByProjectId, []projectV2.Project, []error) {
	var errors []error
	var convertedProjects []projectV2.Project
	projectDefinitions := make(manifest.ProjectDefinitionByProjectId)

	for _, p := range projects {
		adjustedId := adjustProjectId(p.GetId())
		projectDefinition, convertedProject, convertErrors := convertProject(context, environments, adjustedId, p)

		if convertErrors != nil {
			errors = append(errors, convertErrors...)
			continue
		}

		projectDefinitions[projectDefinition.Name] = projectDefinition
		convertedProjects = append(convertedProjects, convertedProject)
	}

	if errors != nil {
		return nil, nil, errors
	}

	return projectDefinitions, convertedProjects, nil
}

var illegalProjectIdCharsRegex = regexp.MustCompile(`[\\/]`)

func adjustProjectId(projectId string) string {
	return illegalProjectIdCharsRegex.ReplaceAllLiteralString(projectId, ".")
}

func convertProject(context *ConverterContext, environments map[string]manifest.EnvironmentDefinition,
	adjustedId string, project projectV1.Project) (manifest.ProjectDefinition, projectV2.Project, []error) {

	convertedConfigs, errors := convertConfigs(&ConfigConvertContext{
		ConverterContext: context,
		ProjectId:        adjustedId,
		V1Apis:           api.NewV1Apis(),
	}, environments, project.GetConfigs())

	if errors != nil {
		return manifest.ProjectDefinition{}, projectV2.Project{}, errors
	}

	dependenciesPerEnvironment := make(map[string][]string)

	for env, apis := range convertedConfigs {
		dependencies := make(map[string]struct{})

		for _, configs := range apis {
			for _, config := range configs {
				// skipped configs have to be ignored
				if config.Skip {
					continue
				}

				for _, ref := range config.References {
					// ignore references on own project
					if ref.Project == config.Coordinate.Project {
						continue
					}

					dependencies[ref.Project] = struct{}{}
				}
			}
		}

		if len(dependencies) == 0 {
			continue
		}

		dependenciesPerEnvironment[env] = mapKeysToSlice(dependencies)
	}

	return manifest.ProjectDefinition{
			Name: adjustedId,
			Path: project.GetId(),
		}, projectV2.Project{
			Id:           adjustedId,
			Configs:      convertedConfigs,
			Dependencies: dependenciesPerEnvironment,
		}, nil
}

func mapKeysToSlice(m map[string]struct{}) []string {
	var result []string

	for k := range m {
		result = append(result, k)
	}

	return result
}

func convertConfigs(context *ConfigConvertContext, environments map[string]manifest.EnvironmentDefinition,
	configs []configV1.Config) (projectV2.ConfigsPerApisPerEnvironments, []error) {

	result := make(projectV2.ConfigsPerApisPerEnvironments)
	var errors []error

	for _, conf := range configs {
		for _, env := range environments {
			if _, found := result[env.Name]; !found {
				result[env.Name] = make(map[string][]configV2.Config)
			}

			convertedConf, err := convertConfig(context, env, conf)

			if err != nil {
				errors = append(errors, err...)
				continue
			}

			apiId := convertedConf.Coordinate.Api
			result[env.Name][apiId] = append(result[env.Name][apiId], convertedConf)
		}
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func convertConfig(context *ConfigConvertContext, environment manifest.EnvironmentDefinition, config configV1.Config) (configV2.Config, []error) {
	var errors []error

	apiId := config.GetApi().GetId()
	convertedTemplatePath := config.GetFilePath()
	apiConversion := api.GetV2ApiId(config.GetApi())
	if apiId != apiConversion {
		log.Info("Converting config '%s' from deprecated API %s to %s", config.GetId(), apiId, apiConversion)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, apiId, apiConversion, 1)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, ".json", "-"+apiId+".json", 1) //ensure modified template paths don't overlap with existing ones
		apiId = apiConversion
	}

	coord := coordinate.Coordinate{
		Project: context.ProjectId,
		Api:     apiId,
		Config:  config.GetId(),
	}

	templ, envParams, listParamIds, errs := convertTemplate(context, config.GetFilePath(), convertedTemplatePath)

	if len(errs) > 0 {
		errors = append(errors, newConvertConfigError(coord, fmt.Sprintf("unable to load template `%s`: %s", config.GetFilePath(), errs)))
	}

	context.KnownListParameterIds = listParamIds

	parameters, references, skip, parameterErrors := convertParameters(context, environment, config)

	if parameterErrors != nil {
		errors = append(errors, parameterErrors...)
	}

	for paramName, param := range envParams {
		if _, found := parameters[paramName]; found {
			errors = append(errors, newConvertConfigError(coord,
				fmt.Sprintf("parameter name collision. automatic environment variable conversion failed. please rename `%s` parameter", paramName)))
			continue
		}

		parameters[paramName] = param
	}

	if errors != nil {
		return configV2.Config{}, errors
	}

	return configV2.Config{
		Template:    templ,
		Coordinate:  coord,
		Group:       environment.Group,
		Environment: environment.Name,
		Parameters:  parameters,
		References:  references,
		Skip:        skip,
	}, nil
}

// TODO make groupable?
type TemplateConversionError struct {
	TemplatePath string
	Reason       string
}

func newTemplateConversionError(templatePath string, reason string) TemplateConversionError {
	return TemplateConversionError{
		TemplatePath: templatePath,
		Reason:       reason,
	}
}

func (e TemplateConversionError) Error() string {
	return fmt.Sprintf("%s: %s", e.TemplatePath, e.Reason)
}

var _ error = (*TemplateConversionError)(nil)

func convertTemplate(context *ConfigConvertContext, currentPath string, writeToPath string) (modifiedTemplate template.Template, envParams map[string]parameter.Parameter, listParameterIds map[string]struct{}, errs []error) {
	data, err := afero.ReadFile(context.Fs, currentPath)

	if err != nil {
		return nil, nil, nil, []error{err}
	}

	templText, environmentParameters, errs := convertEnvVarsReferencesInTemplate(string(data))
	if len(errs) > 0 {
		return nil, nil, nil, errs
	}

	templText, listParameterIds, errs = convertListsInTemplate(templText, currentPath)
	if len(errs) > 0 {
		return nil, nil, nil, errs
	}

	templ := template.CreateTemplateFromString(writeToPath, templText)

	return templ, environmentParameters, listParameterIds, nil
}

func convertEnvVarsReferencesInTemplate(currentTemplate string) (modifiedTemplate string, environmentParameters map[string]parameter.Parameter, errors []error) {
	environmentParameters = map[string]parameter.Parameter{}

	templText := util.EnvVariableRegexPattern.ReplaceAllStringFunc(currentTemplate, func(p string) string {
		envVar := util.TrimToEnvVariableName(p)
		paramName := transformEnvironmentToParamName(envVar)

		if _, found := environmentParameters[paramName]; !found {
			environmentParameters[paramName] = envParam.New(envVar)
		}

		return transformToPropertyAccess(paramName)
	})
	return templText, environmentParameters, errors
}

func transformEnvironmentToParamName(env string) string {
	return fmt.Sprintf("__ENV_%s__", env)
}

func transformToPropertyAccess(property string) string {
	return fmt.Sprintf("{{ .%s }}", property)
}

func convertListsInTemplate(currentTemplate string, currentPath string) (modifiedTemplate string, listParameterIds map[string]struct{}, errors []error) {
	listParameterIds = map[string]struct{}{}

	templText := util.ListVariableRegexPattern.ReplaceAllStringFunc(currentTemplate, func(s string) string {

		fullMatch, fullListMatch, varName, err := util.MatchListVariable(s)
		if err != nil {
			errors = append(errors, newTemplateConversionError(currentPath, err.Error()))
			return ""
		}

		listParameterIds[varName] = struct{}{}
		return strings.Replace(fullMatch, fullListMatch, transformToPropertyAccess(varName), 1)
	})

	return templText, listParameterIds, errors
}

func convertParameters(context *ConfigConvertContext, environment manifest.EnvironmentDefinition,
	config configV1.Config) (map[string]parameter.Parameter, []coordinate.Coordinate, bool, []error) {

	properties := loadPropertiesForEnvironment(environment, config)

	parameters := make(map[string]parameter.Parameter)
	var references []coordinate.Coordinate
	var errors []error
	var skip = false

	for name, value := range properties {
		if name == configV1.SkipConfigDeploymentParameter {
			skipValue, err := parseSkipDeploymentParameter(context, config, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			skip = skipValue
			continue
		}

		if configV1.IsDependency(value) {
			ref, err := parseReference(context, config, name, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			parameters[name] = ref
		} else if _, found := context.KnownListParameterIds[name]; found {
			valueSlice, err := parseListStringToValueSlice(value)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			parameters[name] = &listParam.ListParameter{Values: valueSlice}
		} else if util.IsEnvVariable(value) {
			envVarName := util.TrimToEnvVariableName(value)
			parameters[name] = envParam.New(envVarName)
		} else {
			parameters[name] = &valueParam.ValueParameter{Value: value}
		}

		for _, ref := range parameters[name].GetReferences() {
			references = append(references, ref.Config)
		}
	}

	if errors != nil {
		return parameters, nil, false, errors
	}

	return parameters, references, skip, nil
}

func parseSkipDeploymentParameter(context *ConfigConvertContext, config configV1.Config, value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	location := coordinate.Coordinate{
		Project: context.ProjectId,
		Api:     config.GetApi().GetId(),
		Config:  config.GetId(),
	}

	return false, newConvertConfigError(location, fmt.Sprintf("invalid value for %s: `%s`. allowed values: true, false", configV1.SkipConfigDeploymentParameter, value))
}

func parseReference(context *ConfigConvertContext, config configV1.Config, parameterName string, reference string) (*refParam.ReferenceParameter, error) {
	configId, property, err := configV1.SplitDependency(reference)

	if err != nil {
		return nil, err
	}

	configId = strings.TrimPrefix(configId, "/")

	parts := strings.Split(configId, "/")
	numberOfParts := len(parts)

	if numberOfParts < 3 {
		return nil, newReferenceParserError(context.ProjectId, config, parameterName,
			"not enough parts. please provide <projectId>/<name>/<config>.<property>")
	}

	referencedConfigId := parts[numberOfParts-1]
	referencedApiId := parts[numberOfParts-2]
	projectId := strings.Join(parts[0:numberOfParts-2], ".")

	if !context.V1Apis.IsApi(referencedApiId) {
		return nil, newReferenceParserError(context.ProjectId, config, parameterName, fmt.Sprintf("referenced API '%s' does not exist", referencedApiId))
	}

	currentApiId := api.GetV2ApiId(context.V1Apis[referencedApiId])

	return refParam.New(projectId, currentApiId, referencedConfigId, property), nil
}

func loadPropertiesForEnvironment(environment manifest.EnvironmentDefinition, config configV1.Config) map[string]string {
	result := make(map[string]string)

	for _, propertyName := range []string{config.GetId(), config.GetId() + "." + environment.Group, config.GetId() + "." + environment.Name} {
		properties, found := config.GetProperties()[propertyName]

		if !found {
			continue
		}

		for k, v := range properties {
			result[k] = v
		}
	}

	return result
}

func parseListStringToValueSlice(s string) ([]valueParam.ValueParameter, error) {
	if !util.IsListDefinition(s) && !util.IsSimpleValueDefinition(s) {
		return []valueParam.ValueParameter{}, fmt.Errorf("failed to parse value for list parameter, '%s' is not in expected list format", s)
	}

	var slice []valueParam.ValueParameter
	splitOnColon := strings.Split(s, ",")
	for _, entry := range splitOnColon {
		entry = strings.TrimSpace(entry)
		entry = strings.TrimPrefix(entry, `"`)
		entry = strings.TrimSuffix(entry, `"`)
		if len(entry) > 0 {
			slice = append(slice, valueParam.ValueParameter{Value: entry})
		}
	}
	return slice, nil
}

func convertEnvironments(environments map[string]environmentV1.Environment) map[string]manifest.EnvironmentDefinition {
	result := make(map[string]manifest.EnvironmentDefinition)

	for _, env := range environments {
		var group string

		if env.GetGroup() == "" {
			group = DefaultGroup
		} else {
			group = env.GetGroup()
		}

		result[env.GetId()] = manifest.NewEnvironmentDefinitionFromV1(env, group)
	}

	return result
}
