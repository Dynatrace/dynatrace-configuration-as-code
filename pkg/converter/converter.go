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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/regex"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/list"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	v2template "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/converter/v1environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	projectV1 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v1"
	projectV2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"regexp"
	"slices"
	"strings"
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
	V1Apis                api.APIs
}

type ConvertConfigError struct {
	// Location (coordinate) of the config.Config that failed to be converted
	Location coordinate.Coordinate `json:"location"`
	// Reason describing what went wrong
	Reason string `json:"reason"`
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
	return fmt.Sprintf("cannot convert config %s: %s", e.Location, e.Reason)
}

type ReferenceParseError struct {
	// Location (coordinate) of the config.Config in which a reference failed to be parsed
	Location coordinate.Coordinate `json:"location"`
	// ParameterName is the name of the reference parameter that failed to be parsed
	ParameterName string `json:"parameterName"`
	// Reason describing what went wrong
	Reason string `json:"reason"`
}

func newReferenceParserError(projectId string, config *projectV1.Config, parameterName string, reason string) ReferenceParseError {
	return ReferenceParseError{
		Location: coordinate.Coordinate{
			Project:  projectId,
			Type:     config.GetApi().ID,
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
	projects []projectV1.Project) (manifest.ProjectDefinitionByProjectID, []projectV2.Project, []error) {
	var errors []error
	var convertedProjects []projectV2.Project
	projectDefinitions := make(manifest.ProjectDefinitionByProjectID)

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
		V1Apis:           api.NewV1APIs(),
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
				result[env.Name] = make(map[string][]config.Config)
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

func convertConfig(context *configConvertContext, environment manifest.EnvironmentDefinition, c *projectV1.Config) (config.Config, []error) {
	var errors []error

	apiId := c.GetApi().ID
	convertedTemplatePath := c.GetFilePath()
	apiConversion := api.GetV2ID(c.GetApi())
	if apiId != apiConversion {
		log.Info("Converting config %q from deprecated API %q to %q", c.GetId(), apiId, apiConversion)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, apiId, apiConversion, 1)
		convertedTemplatePath = strings.Replace(convertedTemplatePath, ".json", "-"+apiId+".json", 1) // ensure modified template paths don't overlap with existing ones
		apiId = apiConversion
	} else if deprecatedBy := c.GetApi().DeprecatedBy; deprecatedBy != "" && context.V1Apis.Contains(deprecatedBy) && context.V1Apis[deprecatedBy].NonUniqueName {
		log.Info("Converting config %q from deprecated API %q to config with non-unique-name handling (see https://dt-url.net/non-unique-name-config)", c.GetId(), apiId)
	}

	coord := coordinate.Coordinate{
		Project:  context.ProjectId,
		Type:     apiId,
		ConfigId: c.GetId(),
	}

	templ, envParams, listParamIds, errs := convertTemplate(context, c.GetFilePath(), convertedTemplatePath)

	if len(errs) > 0 {
		errors = append(errors, newConvertConfigError(coord, fmt.Sprintf("unable to load template `%s`: %s", c.GetFilePath(), errs)))
	}

	context.KnownListParameterIds = listParamIds

	parameters, skipParameter, parameterErrors := convertParameters(context, environment, c)

	if parameterErrors != nil {
		errors = append(errors, parameterErrors...)
	}

	// combine the template and config parameters
	for envParamName, envParamVal := range envParams {
		if existingParam, found := parameters[envParamName]; found && !cmp.Equal(envParamVal, existingParam) {
			errors = append(errors, newConvertConfigError(coord,
				fmt.Sprintf("parameter name collision. automatic environment variable conversion failed. please rename `%s` parameter", envParamName)))
			continue
		}

		parameters[envParamName] = envParamVal
	}

	// if the name is missing in the v1 config, create one and log it.
	if _, found := parameters[config.NameParameter]; !found {
		name := c.GetId() + " - monaco-conversion created name"
		parameters[config.NameParameter] = valueParam.New(name)
		log.Info(`Missing name in config "%s/%s/%s". Using name %q.`, c.GetProject(), c.GetType(), c.GetId(), name)
	}

	if errors != nil {
		return config.Config{}, errors
	}

	return config.Config{
		Type:              config.ClassicApiType{Api: apiId},
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
	// TemplatePath is the path to the template JSON file that failed to be converted
	TemplatePath string `json:"templatePath"`
	// Reason describing what went wrong
	Reason string `json:"reason"`
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

func convertTemplate(context *configConvertContext, currentPath string, writeToPath string) (modifiedTemplate v2template.Template, envParams map[string]parameter.Parameter, listParameterIds map[string]struct{}, errs []error) {
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

	return v2template.NewInMemoryTemplateWithPath(writeToPath, temporaryTemplate), environmentParameters, listParameterIds, nil
}

func convertReservedParameters(temporaryTemplate string) string {

	for _, name := range config.ReservedParameterNames {
		r := regexp.MustCompile(fmt.Sprintf(`{{ *\.%s *}}`, name))
		newName := convertReservedParameterNames(name)

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

		newName := convertReservedParameterNames(name)

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
			var refs []parameter.ParameterReference
			envParams := extractEnvParameters(value)

			// if we found just a single parameter without other characters
			// we just convert that to an environment parameter
			if len(envParams) == 1 && value == fmt.Sprintf("{{ .Env.%s }}", envParams[0].Name) {
				parameters[newName] = envParam.New(envParams[0].Name)
				continue
			}

			// else we convert all found environment variables to compound parameter referencing these variables and
			// preserving the format string
			valueWithPlaceHolders := value
			for _, p := range envParams {
				p := p // avoid implicit memory aliasing

				parameterName := fmt.Sprintf("__ENV_%s__", p.Name)
				parameters[parameterName] = &p
				valueWithPlaceHolders = strings.ReplaceAll(strings.ReplaceAll(valueWithPlaceHolders, p.Name, parameterName), ".Env", "")
				// create references
				refs = append(refs, parameter.ParameterReference{
					Config: coordinate.Coordinate{
						Project:  config.GetProject(),
						Type:     config.GetType(),
						ConfigId: config.GetId(),
					},
					Property: parameterName,
				})
			}

			// create compound parameter
			if c, err := compound.New(newName, valueWithPlaceHolders, refs); err != nil {
				errors = append(errors, err)
				continue
			} else {
				parameters[newName] = c
			}
		} else {
			s := removeEscapeChars(value)

			parameters[newName] = &valueParam.ValueParameter{Value: s}
		}
	}

	if errors != nil {
		return parameters, nil, errors
	}

	return parameters, skip, nil
}

// removeEscapeChars turns any manually escaped special characters into just those characters.
// This in combination with the value.ValueParameter's auto-escaping ensures that payloads are constructed
// as expected after conversion.
func removeEscapeChars(value string) string {
	s := strings.ReplaceAll(value, `\\`, `\`)
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}

// convertReservedParametersNames will return a new name for any v1 parameter name that overlaps with a reserved name that is part of toplevel v2 configuration.
// While the 'name' parameter is reserved in v2 as well, it is not converted as it means the same in v1 and v2.
// If the passed paramName does not match a reserved parameter, it will be returned as is
func convertReservedParameterNames(paramName string) string {

	if paramName == config.NameParameter {
		return paramName // 'name' stays the same between v1 and v2
	}

	if slices.Contains(config.ReservedParameterNames, paramName) {
		return paramName + "1"
	}

	return paramName
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
		Type:     config.GetApi().ID,
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
		referencedApiId = config.GetApi().ID
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

	currentApiId := api.GetV2ID(context.V1Apis[referencedApiId])

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

		result[env.GetId()] = newEnvironmentDefinitionFromV1(env, group)
	}

	return result
}

func newEnvironmentDefinitionFromV1(env *v1environment.EnvironmentV1, group string) manifest.EnvironmentDefinition {
	return manifest.EnvironmentDefinition{
		Name:  env.GetId(),
		URL:   newUrlDefinitionFromV1(env),
		Group: group,
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: env.GetTokenName()},
		},
	}
}

func newUrlDefinitionFromV1(env *v1environment.EnvironmentV1) manifest.URLDefinition {
	if regex.IsEnvVariable(env.GetEnvironmentUrl()) {
		// no need to resolve the value for conversion
		return manifest.URLDefinition{
			Type: manifest.EnvironmentURLType,
			Name: regex.TrimToEnvVariableName(env.GetEnvironmentUrl()),
		}
	}

	return manifest.URLDefinition{
		Type:  manifest.ValueURLType,
		Value: strings.TrimSuffix(env.GetEnvironmentUrl(), "/"),
	}
}

func extractEnvParameters(envReference string) []envParam.EnvironmentVariableParameter {
	matches := regex.EnvVariableRegexPattern.FindAllStringSubmatch(envReference, -1)
	parameters := make([]envParam.EnvironmentVariableParameter, 0)
	for _, envVarName := range matches {
		parameters = append(parameters, *envParam.New(envVarName[1]))
	}

	return parameters
}
