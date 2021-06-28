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
	"regexp"
	"strings"

	configv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	configv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	refParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	environmentv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	projectv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

const (
	// default group is used, when a legacy config does not specify a group. All
	// new configs are required to be in a group.
	DefaultGroup = "default"
)

var (
	envVariableRegex = regexp.MustCompile(`{{ *\.Env\.([A-Za-z0-9_-]*) *}}`)
)

type ConverterContext struct {
	Fs afero.Fs
}

type ConfigConvertContext struct {
	*ConverterContext
	ProjectId string
}

type ProjectConverterError struct {
	Project string
	Reason  string
}

func (e *ProjectConverterError) Error() string {
	return fmt.Sprintf("%s: cannot convert project: %s", e.Project, e.Reason)
}

type ConvertConfigError struct {
	Location coordinate.Coordinate
	Reason   string
}

func (e *ConvertConfigError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e *ConvertConfigError) Error() string {
	return fmt.Sprintf("cannot convert config: %s", e.Reason)
}

type ReferenceParserError struct {
	Location      coordinate.Coordinate
	ParameterName string
	Reason        string
}

func (e *ReferenceParserError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e *ReferenceParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse reference: %s",
		e.ParameterName, e.Reason)
}

var (
	_ configErrors.ConfigError = (*ConvertConfigError)(nil)
	_ configErrors.ConfigError = (*ReferenceParserError)(nil)
)

func Convert(context ConverterContext, environments map[string]environmentv1.Environment,
	projects []projectv1.Project) (manifest.Manifest, []projectv2.Project, []error) {
	environmentDefinitions := convertEnvironments(environments)
	projectDefinitions, convertedProjects, errors := convertProjects(&context, environmentDefinitions, projects)

	return manifest.Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
	}, convertedProjects, errors
}

func convertProjects(context *ConverterContext, environments map[string]manifest.EnvironmentDefinition,
	projects []projectv1.Project) (map[string]manifest.ProjectDefinition, []projectv2.Project, []error) {
	var errors []error
	var convertedProjects []projectv2.Project
	projectDefinitions := make(map[string]manifest.ProjectDefinition)

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

func adjustProjectId(projectId string) string {
	return strings.ReplaceAll(projectId, "/", ".")
}

func convertProject(context *ConverterContext, environments map[string]manifest.EnvironmentDefinition,
	adjustedId string, project projectv1.Project) (manifest.ProjectDefinition, projectv2.Project, []error) {

	convertedConfigs, errors := convertConfigs(&ConfigConvertContext{
		ConverterContext: context,
		ProjectId:        adjustedId,
	}, environments, project.GetConfigs())

	if errors != nil {
		return manifest.ProjectDefinition{}, projectv2.Project{}, errors
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
		}, projectv2.Project{
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
	configs []configv1.Config) (projectv2.ConfigsPerApisPerEnvironments, []error) {

	result := make(projectv2.ConfigsPerApisPerEnvironments)
	var errors []error

	for _, conf := range configs {
		for _, env := range environments {
			if _, found := result[env.Name]; !found {
				result[env.Name] = make(map[string][]configv2.Config)
			}

			convertedConf, configErrors := convertConfig(context, env, conf)

			if configErrors != nil {
				errors = append(errors, configErrors...)
				continue
			}

			result[env.Name][conf.GetApi().GetId()] = append(result[env.Name][conf.GetApi().GetId()], convertedConf)
		}
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func convertConfig(context *ConfigConvertContext, environment manifest.EnvironmentDefinition, config configv1.Config) (configv2.Config, []error) {
	var errors []error

	coord := coordinate.Coordinate{
		Project: context.ProjectId,
		Api:     config.GetApi().GetId(),
		Config:  config.GetId(),
	}

	templ, envParams, errs := convertTemplate(context, config.GetFilePath())

	if len(errs) > 0 {
		errors = append(errors, &ConvertConfigError{
			Location: coord,
			Reason:   fmt.Sprintf("unable to load template `%s`: %s", config.GetFilePath(), errs),
		})
	}

	parameters, references, skip, parameterErrors := convertParameters(context, environment, config)

	if parameterErrors != nil {
		errors = append(errors, parameterErrors...)
	}

	for paramName, param := range envParams {
		if _, found := parameters[paramName]; found {
			errors = append(errors, &ConvertConfigError{
				Location: coord,
				Reason: fmt.Sprintf("parameter name collision. automatic environment variable conversion failed. please rename `%s` parameter",
					paramName),
			})
			continue
		}

		parameters[paramName] = param
	}

	if errors != nil {
		return configv2.Config{}, errors
	}

	return configv2.Config{
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
type EnvironmentVariableConverterError struct {
	TemplatePath string
	Reason       string
}

func (e *EnvironmentVariableConverterError) Error() string {
	return fmt.Sprintf("%s: %s", e.TemplatePath, e.Reason)
}

var _ error = (*EnvironmentVariableConverterError)(nil)

func convertTemplate(context *ConfigConvertContext, templatePath string) (template.Template, map[string]parameter.Parameter, []error) {
	data, err := afero.ReadFile(context.Fs, templatePath)

	if err != nil {
		return nil, nil, []error{err}
	}

	environmentParameters := map[string]parameter.Parameter{}
	var errors []error

	templText := envVariableRegex.ReplaceAllStringFunc(string(data), func(p string) string {
		match := envVariableRegex.FindStringSubmatch(p)

		if len(match) != 2 {
			errors = append(errors, &EnvironmentVariableConverterError{
				TemplatePath: templatePath,
				Reason:       fmt.Sprintf("cannot parse environment variable: `%s` seems to be invalid", p),
			})
			return ""
		}

		envVar := match[1]
		paramName := transformEnvironmentToParamName(envVar)

		if _, found := environmentParameters[paramName]; !found {
			environmentParameters[paramName] = &environment.EnvironmentVariableParameter{
				Name:            envVar,
				HasDefaultValue: false,
			}
		}

		return transformToPropertyAccess(paramName)
	})

	templ, err := template.CreateFileBasedTemplateFromString(templatePath, templText)

	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, nil, errors
	}

	return templ, environmentParameters, nil
}

func transformEnvironmentToParamName(env string) string {
	return fmt.Sprintf("__ENV_%s__", env)
}

func transformToPropertyAccess(property string) string {
	return fmt.Sprintf("{{ .%s }}", property)
}

func convertParameters(context *ConfigConvertContext, environment manifest.EnvironmentDefinition,
	config configv1.Config) (map[string]parameter.Parameter, []coordinate.Coordinate, bool, []error) {

	properties := loadPropertiesForEnvironment(environment, config)

	parameters := make(map[string]parameter.Parameter)
	var references []coordinate.Coordinate
	var errors []error
	var skip = false

	for name, value := range properties {
		if name == configv1.SkipConfigDeploymentParameter {
			skipValue, err := parseSkipDeploymentParameter(context, config, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			skip = skipValue
			continue
		}

		if configv1.IsDependency(value) {
			ref, err := parseReference(context, config, name, value)

			if err != nil {
				errors = append(errors, err)
				continue
			}

			parameters[name] = ref
		} else {
			parameters[name] = &valueParam.ValueParameter{
				Value: value,
			}
		}

		for _, ref := range parameters[name].GetReferences() {
			references = append(references, ref.Config)
		}
	}

	if errors != nil {
		return nil, nil, false, errors
	}

	return parameters, references, skip, nil
}

func parseSkipDeploymentParameter(context *ConfigConvertContext, config configv1.Config, value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	return false, &ConvertConfigError{
		Location: coordinate.Coordinate{
			Project: context.ProjectId,
			Api:     config.GetApi().GetId(),
			Config:  config.GetId(),
		},
		Reason: fmt.Sprintf("invalid value for %s: `%s`. allowed values: true, false",
			configv1.SkipConfigDeploymentParameter, value),
	}
}

func parseReference(context *ConfigConvertContext, config configv1.Config, parameterName string, reference string) (*refParam.ReferenceParameter, error) {
	configId, property, err := configv1.SplitDependency(reference)

	if err != nil {
		return nil, err
	}

	configId = strings.TrimPrefix(configId, "/")

	parts := strings.Split(configId, "/")
	numberOfParts := len(parts)

	if numberOfParts < 3 {
		return nil, &ReferenceParserError{
			Location: coordinate.Coordinate{
				Project: context.ProjectId,
				Api:     config.GetApi().GetId(),
				Config:  config.GetId(),
			},
			ParameterName: parameterName,
			Reason:        "not enough parts. please provide <projectId>/<name>/<config>.<property>",
		}
	}

	referencedConfigId := parts[numberOfParts-1]
	apiId := parts[numberOfParts-2]
	projectId := strings.Join(parts[0:numberOfParts-2], ".")

	return &refParam.ReferenceParameter{
		ParameterReference: parameter.ParameterReference{
			Config: coordinate.Coordinate{
				Project: projectId,
				Api:     apiId,
				Config:  referencedConfigId,
			},
			Property: property,
		},
	}, nil
}

func loadPropertiesForEnvironment(environment manifest.EnvironmentDefinition, config configv1.Config) map[string]string {
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

func convertEnvironments(environments map[string]environmentv1.Environment) map[string]manifest.EnvironmentDefinition {
	result := make(map[string]manifest.EnvironmentDefinition)

	for _, env := range environments {
		var group string

		if env.GetGroup() == "" {
			group = DefaultGroup
		} else {
			group = env.GetGroup()
		}

		result[env.GetId()] = manifest.EnvironmentDefinition{
			Name:  env.GetId(),
			Url:   env.GetEnvironmentUrl(),
			Group: group,
			Token: &manifest.EnvironmentVariableToken{
				EnvironmentVariableName: env.GetTokenName(),
			},
		}
	}

	return result
}
