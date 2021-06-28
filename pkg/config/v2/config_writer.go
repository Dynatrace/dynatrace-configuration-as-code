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
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/template"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type WriterContext struct {
	Fs                             afero.Fs
	OutputFolder                   string
	ProjectFolder                  string
	ParametersSerde                map[string]parameter.ParameterSerDe
	UseShortSyntaxForSpecialParams bool
}

type configConverterContext struct {
	*WriterContext
	configFolder string
	config       coordinate.Coordinate
}

type environmentDetails struct {
	group       string
	environment string
}

type detailedConfigConverterContext struct {
	*configConverterContext
	environmentDetails environmentDetails
}

type apiCoordinate struct {
	project string
	api     string
}

type configTemplate struct {
	// absolute path from the monaco project root to the template
	templatePath string

	// content of the template
	content string
}

func WriteConfigs(context *WriterContext, configs []Config) []error {
	definitions, templates, errs := toTopLevelDefinitions(context, configs)

	if len(errs) > 0 {
		return errs
	}

	var writeErrors []error

	for apiCoord, definition := range definitions {
		err := writeTopLevelDefinitionToDisk(context, apiCoord, definition)

		if err != nil {
			writeErrors = append(writeErrors, err)
		}
	}

	writeErrors = append(writeErrors, writeTemplates(context, templates)...)

	if len(writeErrors) > 0 {
		return writeErrors
	}

	return nil
}

func writeTemplates(context *WriterContext, templates []configTemplate) (errors []error) {
	for _, t := range templates {
		fullTemplatePath := filepath.Join(context.OutputFolder, t.templatePath)
		templateDir := filepath.Dir(fullTemplatePath)

		err := context.Fs.MkdirAll(templateDir, 0777)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = afero.WriteFile(context.Fs, fullTemplatePath, []byte(t.content), 0664)

		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func toTopLevelDefinitions(context *WriterContext, configs []Config) (map[apiCoordinate]topLevelDefinition, []configTemplate, []error) {
	configsPerCoordinate := groupConfigs(configs)

	var errors []error
	result := map[apiCoordinate]topLevelDefinition{}

	configsPerApi := map[apiCoordinate][]topLevelConfigDefinition{}
	knownTemplates := map[string]struct{}{}
	var configTemplates []configTemplate

	for coord, confs := range configsPerCoordinate {
		configContext := &configConverterContext{
			WriterContext: context,
			configFolder:  filepath.Join(context.ProjectFolder, coord.Api),
			config:        coord,
		}

		definition, templates, errs := toTopLevelConfigDefinition(configContext, confs)

		if len(errs) > 0 {
			errors = append(errors, errs...)
			continue
		}

		apiCoord := apiCoordinate{
			project: coord.Project,
			api:     coord.Api,
		}

		configsPerApi[apiCoord] = append(configsPerApi[apiCoord], definition)

		for _, t := range templates {
			if _, found := knownTemplates[t.templatePath]; found {
				continue
			}

			configTemplates = append(configTemplates, t)
			knownTemplates[t.templatePath] = struct{}{}
		}
	}

	if len(errors) > 0 {
		return nil, nil, errors
	}

	for apiCoord, confs := range configsPerApi {
		result[apiCoord] = topLevelDefinition{
			Configs: confs,
		}
	}

	return result, configTemplates, nil
}

func writeTopLevelDefinitionToDisk(context *WriterContext, apiCoord apiCoordinate, definiton topLevelDefinition) error {
	definitionYaml, err := yaml.Marshal(definiton)

	if err != nil {
		return err
	}

	targetConfigFile := filepath.Join(context.OutputFolder, context.ProjectFolder, apiCoord.api, "config.yaml")

	err = context.Fs.MkdirAll(filepath.Dir(targetConfigFile), 0777)

	if err != nil {
		return err
	}

	err = afero.WriteFile(context.Fs, targetConfigFile, definitionYaml, 0664)

	if err != nil {
		return err
	}

	return nil
}

func toTopLevelConfigDefinition(context *configConverterContext, configs []Config) (topLevelConfigDefinition, []configTemplate, []error) {
	configDefinitions, templates, errs := toConfigDefinitions(context, configs)

	if len(errs) > 0 {
		return topLevelConfigDefinition{}, nil, errs
	}

	groupedDefinitionsByGroup := groupByGroups(configDefinitions)

	var groupOverrides []extendedConfigDefinition
	var environmentOverrides []extendedConfigDefinition

	for group, definitions := range groupedDefinitionsByGroup {
		base, reduced := extractCommonBase(definitions)

		if base != nil {
			groupOverrides = append(groupOverrides, extendedConfigDefinition{
				configDefinition: *base,
				group:            group,
				environment:      "",
			})
		}

		environmentOverrides = append(environmentOverrides, reduced...)
	}

	baseConfig, reducedGroupOverrides := extractCommonBase(groupOverrides)

	var config configDefinition
	var groupOverrideConfigs []groupOverride
	var environmentOverrideConfigs []environmentOverride

	if baseConfig != nil {
		config = *baseConfig
	}

	for _, conf := range reducedGroupOverrides {
		groupOverrideConfigs = append(groupOverrideConfigs, groupOverride{
			Group:    conf.group,
			Override: conf.configDefinition,
		})
	}

	for _, conf := range environmentOverrides {
		environmentOverrideConfigs = append(environmentOverrideConfigs, environmentOverride{
			Environment: conf.environment,
			Override:    conf.configDefinition,
		})
	}

	return topLevelConfigDefinition{
		Id:                   context.config.Config,
		Config:               config,
		GroupOverrides:       groupOverrideConfigs,
		EnvironmentOverrides: environmentOverrideConfigs,
	}, templates, nil
}

func groupByGroups(configs []extendedConfigDefinition) map[string][]extendedConfigDefinition {

	result := make(map[string][]extendedConfigDefinition)

	for _, c := range configs {
		result[c.group] = append(result[c.group], c)
	}

	return result
}

func extractCommonBase(configs []extendedConfigDefinition) (*configDefinition, []extendedConfigDefinition) {
	switch len(configs) {
	case 0:
		return nil, nil
	case 1:
		return &configs[0].configDefinition, nil
	}

	checkResult := testForSameProperties(configs)
	sharedParam := extractSharedParameters(configs)

	// TODO refactor this monstrosity
	if len(sharedParam) == 0 && (!checkResult.foundName || !checkResult.shareName) &&
		(!checkResult.foundTemplate || !checkResult.shareTemplate) &&
		(!checkResult.foundSkip || !checkResult.shareSkip) {
		return nil, configs
	}

	configDefinitionResult := createCommonConfigDefinition(checkResult, sharedParam)
	var definitions []extendedConfigDefinition

	for _, conf := range configs {
		reducedConf := createConfigDefinitionWithoutSharedValues(conf, checkResult, sharedParam)

		if reducedConf != nil {
			definitions = append(definitions, extendedConfigDefinition{
				configDefinition: *reducedConf,
				group:            conf.group,
				environment:      conf.environment,
			})
		}
	}

	return configDefinitionResult, definitions
}

func createConfigDefinitionWithoutSharedValues(toReduce extendedConfigDefinition, checkResult propertyCheckResult,
	sharedParameters map[string]configParameter) *configDefinition {
	var allParametersShared bool = true
	reducedParameters := make(map[string]configParameter)

	for k, v := range toReduce.Parameters {
		if _, found := sharedParameters[k]; !found {
			allParametersShared = false
			reducedParameters[k] = v
		}
	}

	if allParametersShared && checkResult.shareName &&
		checkResult.shareSkip && checkResult.shareTemplate {
		return nil
	}

	result := &configDefinition{
		Parameters: reducedParameters,
	}

	if !checkResult.shareName {
		result.Name = toReduce.Name
	}

	if !checkResult.shareTemplate {
		result.Template = toReduce.Template
	}

	if !checkResult.shareSkip {
		result.Skip = toReduce.Skip
	}

	return result
}

func createCommonConfigDefinition(checkResult propertyCheckResult, sharedParameters map[string]configParameter) *configDefinition {
	result := &configDefinition{}

	if checkResult.foundName || checkResult.shareName {
		result.Name = checkResult.name
	}

	if checkResult.foundTemplate || checkResult.shareTemplate {
		result.Template = checkResult.template
	}

	if checkResult.foundSkip || checkResult.shareSkip {
		result.Skip = checkResult.skip
	}

	if len(sharedParameters) > 0 {
		result.Parameters = sharedParameters
	}

	return result
}

func extractSharedParameters(configs []extendedConfigDefinition) map[string]configParameter {
	result := make(map[string]configParameter)
	startParams := configs[0].Parameters

ParamLoop:
	for name, val := range startParams {
		for i := 1; i < len(configs); i++ {
			conf := configs[i]
			paramVal := conf.Parameters[name]

			if !reflect.DeepEqual(val, paramVal) {
				// TODO should probably be refactored (loops with labels
				// are kinda a code smell)
				continue ParamLoop
			}
		}

		result[name] = val
	}

	return result
}

type propertyCheckResult struct {
	shareName bool
	foundName bool
	name      configParameter

	shareTemplate bool
	foundTemplate bool
	template      string

	shareSkip bool
	foundSkip bool
	skip      interface{}
}

func testForSameProperties(configs []extendedConfigDefinition) propertyCheckResult {
	name := configs[0].Name
	template := configs[0].Template
	skip := configs[0].Skip

	var (
		sameName,
		sameTemplate,
		sameSkip = true, true, true
	)

	for _, c := range configs {
		sameName = sameName && reflect.DeepEqual(name, c.Name)
		sameTemplate = sameTemplate && template == c.Template
		sameSkip = sameSkip && (reflect.DeepEqual(skip, c.Skip) ||
			(skip == nil && c.Skip == false) ||
			(skip == false && c.Skip == nil))
	}

	if !sameName {
		name = nil
	}

	if !sameTemplate {
		template = ""
	}

	if !sameSkip {
		skip = nil
	}

	return propertyCheckResult{
		shareName: sameName,
		foundName: name != nil || !sameName,
		name:      name,

		shareTemplate: sameTemplate,
		foundTemplate: template != "" || !sameTemplate,
		template:      template,

		shareSkip: sameSkip,
		foundSkip: skip != nil || !sameSkip,
		skip:      skip,
	}
}

type extendedConfigDefinition struct {
	configDefinition
	group       string
	environment string
}

func toConfigDefinitions(context *configConverterContext, configs []Config) ([]extendedConfigDefinition, []configTemplate, []error) {
	var errors []error
	result := make([]extendedConfigDefinition, 0, len(configs))

	var templates []configTemplate

	for _, c := range configs {
		definition, template, errs := toConfigDefinition(context, c)

		if len(errs) > 0 {
			errors = append(errors, errs...)
			continue
		}

		templates = append(templates, template)

		result = append(result, extendedConfigDefinition{
			configDefinition: definition,
			group:            c.Group,
			environment:      c.Environment,
		})
	}

	if len(errors) > 0 {
		return nil, nil, errors
	}

	return result, templates, nil
}

func toConfigDefinition(context *configConverterContext, config Config) (configDefinition, configTemplate, []error) {
	var errors []error
	detailedContext := detailedConfigConverterContext{
		configConverterContext: context,
		environmentDetails: environmentDetails{
			group:       config.Group,
			environment: config.Environment,
		},
	}
	nameParam, err := parseNameParameter(&detailedContext, config)

	if err != nil {
		errors = append(errors, err)
	}

	params, errs := convertParameters(&detailedContext, config.Parameters)

	errors = append(errors, errs...)

	configTemplatePath, template, err := extractTemplate(&detailedContext, config)

	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return configDefinition{}, configTemplate{}, errors
	}

	return configDefinition{
		Name:       nameParam,
		Parameters: params,
		Template:   configTemplatePath,
		Skip:       config.Skip,
	}, template, nil
}

func extractTemplate(context *detailedConfigConverterContext, config Config) (string, configTemplate, error) {
	switch templ := config.Template.(type) {
	case template.FileBasedTemplate:
		path, err := filepath.Rel(context.configFolder, filepath.Clean(templ.FilePath()))

		if err != nil {
			return "", configTemplate{}, err
		}

		return path, configTemplate{
			templatePath: templ.FilePath(),
			content:      templ.Content(),
		}, nil
	case template.Template:
		sanitizedName := util.SanitizeName(templ.Name())

		return sanitizedName, configTemplate{
			templatePath: filepath.Join(context.configFolder, sanitizedName),
			content:      templ.Content(),
		}, nil
	}

	// this should never happen
	return "", configTemplate{}, errors.New("unknown template type")
}

func convertParameters(context *detailedConfigConverterContext, parameters Parameters) (map[string]configParameter, []error) {
	var errors []error
	result := make(map[string]configParameter)

	for name, param := range parameters {
		// ignore NameParameter as it is handled in a special way
		if name == NameParameter {
			continue
		}

		parsed, err := toParameterDefinition(context, name, param)

		if err != nil {
			errors = append(errors, err)
			continue
		}

		result[name] = parsed
	}

	if len(errors) > 0 {
		return nil, errors
	}

	return result, nil
}

func parseNameParameter(context *detailedConfigConverterContext, config Config) (configParameter, error) {
	nameParam, found := config.Parameters[NameParameter]

	if !found {
		return nil, fmt.Errorf("%s: `name` paramter missing",
			config.Coordinate.ToString())
	}

	return toParameterDefinition(context, NameParameter, nameParam)
}

func toParameterDefinition(context *detailedConfigConverterContext, parameterName string,
	param parameter.Parameter) (configParameter, error) {

	if context.UseShortSyntaxForSpecialParams && isSpecialParameter(param) {
		return toSpecialParameterDefinition(context, parameterName, param)
	}

	serde, found := context.ParametersSerde[param.GetType()]

	if !found {
		return nil, fmt.Errorf("%s:%s: no serde found for type `%s`",
			context.config.ToString(), parameterName, param.GetType())
	}

	result, err := serde.Serializer(parameter.ParameterWriterContext{
		Coordinate:    context.config,
		Group:         context.environmentDetails.group,
		Environment:   context.environmentDetails.environment,
		ParameterName: parameterName,
		Parameter:     param,
	})

	if err != nil {
		return nil, err
	}

	result["type"] = param.GetType()

	return result, nil
}

func isSpecialParameter(param parameter.Parameter) bool {
	return param.GetType() == value.ValueParameterType
}

func toSpecialParameterDefinition(context *detailedConfigConverterContext, parameterName string,
	param parameter.Parameter) (configParameter, error) {
	switch param.GetType() {
	case value.ValueParameterType:
		valueParam, ok := param.(*value.ValueParameter)

		if !ok {
			return nil, fmt.Errorf("%s:%s: parameter of type `%s` is no value param!", context.config.ToString(), parameterName, param.GetType())
		}

		switch valueParam.Value.(type) {
		// map/array values need special handling to not collide with other paramters
		case map[string]interface{}, []interface{}:
			result, err := context.ParametersSerde[param.GetType()].Serializer(parameter.ParameterWriterContext{
				Coordinate:    context.config,
				Group:         context.environmentDetails.group,
				Environment:   context.environmentDetails.environment,
				ParameterName: parameterName,
				Parameter:     param,
			})

			if err != nil {
				return nil, err
			}

			result["type"] = valueParam.GetType()

			return result, nil
		default:
			return valueParam.Value, nil
		}
	}

	return nil, fmt.Errorf("%s:%s: unknown special type `%s`", context.config.ToString(), parameterName, param.GetType())
}

func groupConfigs(configs []Config) map[coordinate.Coordinate][]Config {
	result := make(map[coordinate.Coordinate][]Config)

	for _, c := range configs {
		result[c.Coordinate] = append(result[c.Coordinate], c)
	}

	return result
}
