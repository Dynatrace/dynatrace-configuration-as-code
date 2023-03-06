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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/regex"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/list"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/converter/v1environment"
	projectV1 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v1"
	"regexp"
	"strings"

	configV2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/environment"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	projectV2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

const (
	// DefaultGroup is used, when a legacy config does not specify a group. All
	// new configs are required to be in a group.
	DefaultGroup = "default"
)

type ConverterContext struct {
	Fs afero.Fs

	ResolveSkip bool
}

type configConvertContext struct {
	*ConverterContext
	ProjectId             string
	KnownListParameterIds map[string]struct{}
	V1Apis                api.ApiMap
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

type ReferenceParseError struct {
	Location      coordinate.Coordinate
	ParameterName string
	Reason        string
}

func newReferenceParserError(projectId string, config *projectV1.Config, parameterName string, reason string) ReferenceParseError {
	return ReferenceParseError{
		Location: coordinate.Coordinate{
			Project:  projectId,
			Type:     config.GetApi().GetId(),
			ConfigId: config.GetId(),
		},
		ParameterName: parameterName,
		Reason:        reason,
	}
}

func (e ReferenceParseError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e ReferenceParseError) Error() string {
	return fmt.Sprintf("%s: cannot parse reference: %s",
		e.ParameterName, e.Reason)
}

var (
	_ configErrors.ConfigError = (*ConvertConfigError)(nil)
	_ configErrors.ConfigError = (*ReferenceParseError)(nil)
)

// Convert takes v1 environments and projects and converts them into a v2 manifest and projects
func Convert(context ConverterContext, environments map[string]*v1environment.EnvironmentV1,
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

	convertedConfigs, errors := convertConfigs(&configConvertContext{
		ConverterContext: context,
		ProjectId:        adjustedId,
		V1Apis:           api.NewV1Apis(),
	}, environments, project.GetConfigs())

	if errors != nil {
		return manifest.ProjectDefinition{}, projectV2.Project{}, errors
	}

	return manifest.ProjectDefinition{
			Name: adjustedId,
			Path: project.GetId(),
		}, projectV2.Project{
			Id:      adjustedId,
			Configs: convertedConfigs,
		}, nil
}

func convertConfigs(context *configConvertContext, environments map[string]manifest.EnvironmentDefinition,
	configs []*projectV1.Config) (projectV2.ConfigsPerTypePerEnvironments, []error) {

	result := make(projectV2.ConfigsPerTypePerEnvironments)
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

			configType := convertedConf.Coordinate.Type
			result[env.Name][configType] = append(result[env.Name][configType], convertedConf)
		}
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func convertConfig(context *configConvertContext, environment manifest.EnvironmentDefinition, config *projectV1.Config) (configV2.Config, []error) {
	var errors []error

	apiId := config.GetApi().GetId()
	convertedTemplatePath := config.GetFilePath()
	apiConversion := api.GetV2ApiId(config.GetApi())
	if apiId != apiConversion {
		log.Info("Converting config %q from deprecated API %q to %q", config.GetId(), apiId, apiConversion)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, apiId, apiConversion, 1)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, ".json", "-"+apiId+".json", 1) //ensure modified template paths don't overlap with existing ones
		apiId = apiConversion
	} else if deprecatedBy := config.GetApi().DeprecatedBy(); deprecatedBy != "" && context.V1Apis.Contains(deprecatedBy) && context.V1Apis[deprecatedBy].IsNonUniqueNameApi() {
		log.Info("Converting config %q from deprecated API %q to config with non-unique-name handling (see https://dt-url.net/non-unique-name-config)", config.GetId(), apiId)
	}

	coord := coordinate.Coordinate{
		Project:  context.ProjectId,
		Type:     apiId,
		ConfigId: config.GetId(),
	}

	templ, envParams, listParamIds, errs := convertTemplate(context, config.GetFilePath(), convertedTemplatePath)

	if len(errs) > 0 {
		errors = append(errors, newConvertConfigError(coord, fmt.Sprintf("unable to load template `%s`: %s", config.GetFilePath(), errs)))
	}

	context.KnownListParameterIds = listParamIds

	parameters, skipParameter, parameterErrors := convertParameters(context, environment, config)

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

	// if the name is missing in the v1 config, create one and log it.
	if _, found := parameters[configV2.NameParameter]; !found {
		name := config.GetId() + " - monaco-conversion created name"
		parameters[configV2.NameParameter] = valueParam.New(name)
		log.Info(`Missing name in config "%s/%s/%s". Using name %q.`, config.GetProject(), config.GetType(), config.GetId(), name)
	}

	if errors != nil {
		return configV2.Config{}, errors
	}

	return configV2.Config{
		Template:          templ,
		Coordinate:        coord,
		Group:             environment.Group,
		Environment:       environment.Name,
		Parameters:        parameters,
		Skip:              false,
		SkipForConversion: skipParameter,
	}, nil
}

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

func convertTemplate(context *configConvertContext, currentPath string, writeToPath string) (modifiedTemplate template.Template, envParams map[string]parameter.Parameter, listParameterIds map[string]struct{}, errs []error) {
	data, err := afero.ReadFile(context.Fs, currentPath)
	if err != nil {
		return nil, nil, nil, []error{err}
	}

	temporaryTemplate, environmentParameters := convertEnvVarsReferencesInTemplate(string(data))
	temporaryTemplate = convertReservedParameters(temporaryTemplate)
	temporaryTemplate, listParameterIds, errs = convertListsInTemplate(temporaryTemplate, currentPath)
	if len(errs) > 0 {
		return nil, nil, nil, errs
	}

	return template.CreateTemplateFromString(writeToPath, temporaryTemplate), environmentParameters, listParameterIds, nil
}

func convertReservedParameters(temporaryTemplate string) string {

	for _, name := range configV2.ReservedParameterNames {
		if name == configV2.NameParameter {
			continue // name stays the same in the template
		}

		r := regexp.MustCompile(fmt.Sprintf(`{{ *\.%s *}}`, name))
		newName := convertedParameterName(name)

		temporaryTemplate = r.ReplaceAllString(temporaryTemplate, fmt.Sprintf("{{ .%s }}", newName))

	}

	return temporaryTemplate
}

func convertEnvVarsReferencesInTemplate(currentTemplate string) (modifiedTemplate string, environmentParameters map[string]parameter.Parameter) {
	environmentParameters = map[string]parameter.Parameter{}

	templText := regex.EnvVariableRegexPattern.ReplaceAllStringFunc(currentTemplate, func(p string) string {
		envVar := regex.TrimToEnvVariableName(p)
		paramName := transformEnvironmentToParamName(envVar)

		if _, found := environmentParameters[paramName]; !found {
			environmentParameters[paramName] = envParam.New(envVar)
		}

		return transformToPropertyAccess(paramName)
	})
	return templText, environmentParameters
}

func transformEnvironmentToParamName(env string) string {
	return fmt.Sprintf("__ENV_%s__", env)
}

func transformToPropertyAccess(property string) string {
	return fmt.Sprintf("{{ .%s }}", property)
}

func convertListsInTemplate(currentTemplate string, currentPath string) (modifiedTemplate string, listParameterIds map[string]struct{}, errors []error) {
	listParameterIds = map[string]struct{}{}

	templText := regex.ListVariableRegexPattern.ReplaceAllStringFunc(currentTemplate, func(s string) string {

		fullMatch, fullListMatch, varName, err := regex.MatchListVariable(s)
		if err != nil {
			errors = append(errors, newTemplateConversionError(currentPath, err.Error()))
			return ""
		}

		listParameterIds[varName] = struct{}{}
		return strings.Replace(fullMatch, fullListMatch, transformToPropertyAccess(varName), 1)
	})

	return templText, listParameterIds, errors
}

func convertParameters(context *configConvertContext, environment manifest.EnvironmentDefinition,
	config *projectV1.Config) (map[string]parameter.Parameter, parameter.Parameter, []error) {

	properties := loadPropertiesForEnvironment(environment, config)

	parameters := make(map[string]parameter.Parameter)
	var errors []error
	var skip parameter.Parameter

	for name, value := range properties {
		if name == projectV1.SkipConfigDeploymentParameter {
			skipParameter, err := parseSkipDeploymentParameter(context, config, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			skip = skipParameter
			continue
		}

		newName := convertedParameterName(name)

		if projectV1.IsDependency(value) {
			ref, err := parseReference(context, config, name, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			parameters[newName] = ref
		} else if _, found := context.KnownListParameterIds[name]; found {
			valueSlice, err := parseListStringToValueSlice(value)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			parameters[newName] = &listParam.ListParameter{Values: valueSlice}
		} else if regex.IsEnvVariable(value) {
			envVarName := regex.TrimToEnvVariableName(value)
			parameters[newName] = envParam.New(envVarName)
		} else {
			parameters[newName] = &valueParam.ValueParameter{Value: value}
		}
	}

	if errors != nil {
		return parameters, nil, errors
	}

	return parameters, skip, nil
}

func convertedParameterName(name string) string {

	if slices.Contains(configV2.ReservedParameterNames, name) {
		return name + "1"
	}

	return name
}

func parseSkipDeploymentParameter(context *configConvertContext, config *projectV1.Config, value string) (parameter.Parameter, error) {
	switch strings.ToLower(value) {
	case "true":
		return valueParam.New(true), nil
	case "false":
		return valueParam.New(false), nil
	}

	if regex.IsEnvVariable(value) {
		envVarName := regex.TrimToEnvVariableName(value)

		return envParam.New(envVarName), nil
	}

	location := coordinate.Coordinate{
		Project:  context.ProjectId,
		Type:     config.GetApi().GetId(),
		ConfigId: config.GetId(),
	}

	return nil, newConvertConfigError(location, fmt.Sprintf("invalid value for %s: `%s`. allowed values: true, false", projectV1.SkipConfigDeploymentParameter, value))
}

func parseReference(context *configConvertContext, config *projectV1.Config, parameterName string, reference string) (*refParam.ReferenceParameter, error) {
	configId, property, err := projectV1.SplitDependency(reference)

	if err != nil {
		return nil, err
	}

	configId = strings.TrimPrefix(configId, "/")
	parts := strings.Split(configId, "/")

	var projectId, referencedApiId, referencedConfigId string

	switch numberOfParts := len(parts); numberOfParts {
	case 0:
		return nil, newReferenceParserError(context.ProjectId, config, parameterName,
			"wrong reference format. Please provide '<projectId>/<name>/<config>.<property>' for referencing another project, '<name>/<config>.<property>' for referencing within the same project, or <config>.<property> for referencing within the same config")

	case 1:
		projectId = context.ProjectId
		referencedApiId = config.GetApi().GetId()
		referencedConfigId = parts[0]

	case 2:
		projectId = context.ProjectId
		referencedApiId = parts[0]
		referencedConfigId = parts[1]

	default:
		projectId = strings.Join(parts[0:numberOfParts-2], ".")
		referencedApiId = parts[numberOfParts-2]
		referencedConfigId = parts[numberOfParts-1]
	}

	if !context.V1Apis.Contains(referencedApiId) {
		return nil, newReferenceParserError(context.ProjectId, config, parameterName, fmt.Sprintf("referenced API '%s' does not exist", referencedApiId))
	}

	currentApiId := api.GetV2ApiId(context.V1Apis[referencedApiId])

	return refParam.New(projectId, currentApiId, referencedConfigId, property), nil
}

func loadPropertiesForEnvironment(environment manifest.EnvironmentDefinition, config *projectV1.Config) map[string]string {
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
	if !regex.IsListDefinition(s) && !regex.IsSimpleValueDefinition(s) {
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

func convertEnvironments(environments map[string]*v1environment.EnvironmentV1) map[string]manifest.EnvironmentDefinition {
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
