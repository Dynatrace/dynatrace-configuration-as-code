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

package writer

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	mystrings "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configError "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/internal/persistence"
)

type WriterContext struct {
	Fs              afero.Fs
	OutputFolder    string
	ProjectFolder   string
	ParametersSerde map[string]parameter.ParameterSerDe
}

type serializerContext struct {
	*WriterContext
	configFolder string
	config       coordinate.Coordinate
}

type environmentDetails struct {
	group       string
	environment string
}

type detailedSerializerContext struct {
	*serializerContext
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

func WriteConfigs(context *WriterContext, configs []config.Config) []error {
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
			errors = append(errors, newConfigWriterError(context, err))
			continue
		}

		err = afero.WriteFile(context.Fs, fullTemplatePath, []byte(t.content), 0664)

		if err != nil {
			errors = append(errors, newConfigWriterError(context, err))
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

func toTopLevelDefinitions(context *WriterContext, configs []config.Config) (map[apiCoordinate]persistence.TopLevelDefinition, []configTemplate, []error) {
	configsPerCoordinate := groupConfigs(configs)

	var errs []error
	result := map[apiCoordinate]persistence.TopLevelDefinition{}

	configsPerApi := map[apiCoordinate][]persistence.TopLevelConfigDefinition{}
	knownTemplates := map[string]struct{}{}
	var configTemplates []configTemplate

	for coord, confs := range configsPerCoordinate {
		sanitizedType := mystrings.Sanitize(coord.extendedType)
		configContext := &serializerContext{
			WriterContext: context,
			configFolder:  filepath.Join(context.ProjectFolder, sanitizedType),
			config:        coord.Coordinate,
		}

		definition, templates, convertErrs := toTopLevelConfigDefinition(configContext, confs)

		if len(convertErrs) > 0 {
			errs = append(errs, convertErrs...)
			continue
		}

		apiCoord := apiCoordinate{
			project: coord.Project,
			api:     coord.extendedType,
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

	if len(errs) > 0 {
		return nil, nil, errs
	}

	for apiCoord, confs := range configsPerApi {
		result[apiCoord] = persistence.TopLevelDefinition{
			Configs: confs,
		}
	}

	return result, configTemplates, nil
}

func byConfigId(a, b persistence.TopLevelConfigDefinition) int {
	return strings.Compare(a.Id, b.Id)
}

func writeTopLevelDefinitionToDisk(context *WriterContext, apiCoord apiCoordinate, definition persistence.TopLevelDefinition) error {
	// sort configs so that they are stable within a config file
	slices.SortFunc(definition.Configs, byConfigId)

	definitionYaml, err := yaml.Marshal(definition)
	if err != nil {
		return newConfigWriterError(context, err)
	}

	sanitizedApi := mystrings.Sanitize(apiCoord.api)
	targetConfigFile := filepath.Join(context.OutputFolder, context.ProjectFolder, sanitizedApi, "config.yaml")

	err = context.Fs.MkdirAll(filepath.Dir(targetConfigFile), 0777)

	if err != nil {
		return newConfigWriterError(context, err)
	}

	err = afero.WriteFile(context.Fs, targetConfigFile, definitionYaml, 0664)

	if err != nil {
		return newConfigWriterError(context, err)
	}

	return nil
}

func toTopLevelConfigDefinition(context *serializerContext, configs []config.Config) (persistence.TopLevelConfigDefinition, []configTemplate, []error) {
	configDefinitions, templates, errs := toConfigDefinitions(context, configs)

	if len(errs) > 0 {
		return persistence.TopLevelConfigDefinition{}, nil, errs
	}

	groupedDefinitionsByGroup := groupByGroups(configDefinitions)

	var groupOverrides []extendedConfigDefinition
	var environmentOverrides []extendedConfigDefinition

	for group, definitions := range groupedDefinitionsByGroup {
		base, reduced := extractCommonBase(definitions)

		if base != nil {
			groupOverrides = append(groupOverrides, extendedConfigDefinition{
				ConfigDefinition: *base,
				group:            group,
			})
		}

		environmentOverrides = append(environmentOverrides, reduced...)
	}

	baseConfig, reducedGroupOverrides := extractCommonBase(groupOverrides)

	var config persistence.ConfigDefinition
	var groupOverrideConfigs []persistence.GroupOverride
	var environmentOverrideConfigs []persistence.EnvironmentOverride

	if baseConfig != nil {
		config = *baseConfig
	}

	for _, conf := range reducedGroupOverrides {
		groupOverrideConfigs = append(groupOverrideConfigs, persistence.GroupOverride{
			Group:    conf.group,
			Override: conf.ConfigDefinition,
		})
	}

	for _, conf := range environmentOverrides {
		environmentOverrideConfigs = append(environmentOverrideConfigs, persistence.EnvironmentOverride{
			Environment: conf.environment,
			Override:    conf.ConfigDefinition,
		})
	}

	// We need to extract the configType from the original configs.
	// Since they all should have the same configType (they have all the same coordinate), we can take any one.
	ct, err := extractConfigType(context, configs[0])
	if err != nil {
		return persistence.TopLevelConfigDefinition{}, nil, []error{
			fmtDetailedConfigWriterError(context, fmt.Errorf("failed to extract config type: %w", err)),
		}
	}

	return persistence.TopLevelConfigDefinition{
		Id:                   context.config.ConfigId,
		Config:               config,
		Type:                 ct,
		GroupOverrides:       groupOverrideConfigs,
		EnvironmentOverrides: environmentOverrideConfigs,
	}, templates, nil
}

func extractConfigType(context *serializerContext, cfg config.Config) (persistence.TypeDefinition, error) {
	ttype := persistence.TypeDefinition{
		Type: cfg.Type,
	}

	switch cfg.Type.(type) {
	case config.SettingsType:
		serializedScope, err := getSerializedParam(context, cfg, config.ScopeParameter, true)
		if err != nil {
			return persistence.TypeDefinition{}, err
		}
		ttype.Scope = serializedScope

		serializedInsertAfter, err := getSerializedParam(context, cfg, config.InsertAfterParameter, false)
		if err != nil {
			return persistence.TypeDefinition{}, err
		}
		ttype.InsertAfter = serializedInsertAfter

	case config.ClassicApiType:
		// TODO: Check if API is a subpath API and handle it accordingly.
		// for now just check if we can exract a scope, and if we can, use it

		serializedScope, err := getSerializedParam(context, cfg, config.ScopeParameter, true)
		if err == nil {
			ttype.Scope = serializedScope
		}
	}
	return ttype, nil
}

func getSerializedParam(context *serializerContext, cfg config.Config, paramName string, required bool) (persistence.ConfigParameter, error) {
	param, found := cfg.Parameters[paramName]
	if !found {
		if required {
			return nil, fmtDetailedConfigWriterError(context, fmt.Errorf("'%s' parameter not found. This is likely a bug", paramName))
		}
		return nil, nil
	}

	serializedScope, err := toParameterDefinition(&detailedSerializerContext{
		serializerContext: context,
	}, paramName, param)
	if err != nil {
		return nil, fmtDetailedConfigWriterError(context, fmt.Errorf("failed to serialize '%s' parameter: %w", paramName, err))
	}
	return serializedScope, nil
}

func groupByGroups(configs []extendedConfigDefinition) map[string][]extendedConfigDefinition {

	result := make(map[string][]extendedConfigDefinition)

	for _, c := range configs {
		result[c.group] = append(result[c.group], c)
	}

	return result
}

func extractCommonBase(configs []extendedConfigDefinition) (*persistence.ConfigDefinition, []extendedConfigDefinition) {
	switch len(configs) {
	case 0:
		return nil, nil
	case 1:
		return &configs[0].ConfigDefinition, nil
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
				ConfigDefinition: *reducedConf,
				group:            conf.group,
				environment:      conf.environment,
			})
		}
	}

	return configDefinitionResult, definitions
}

func createConfigDefinitionWithoutSharedValues(toReduce extendedConfigDefinition, checkResult propertyCheckResult,
	sharedParameters map[string]persistence.ConfigParameter) *persistence.ConfigDefinition {
	allParametersShared := true
	reducedParameters := make(map[string]persistence.ConfigParameter)

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

	result := &persistence.ConfigDefinition{
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

func createCommonConfigDefinition(checkResult propertyCheckResult, sharedParameters map[string]persistence.ConfigParameter) *persistence.ConfigDefinition {
	result := &persistence.ConfigDefinition{}

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

func extractSharedParameters(configs []extendedConfigDefinition) map[string]persistence.ConfigParameter {
	result := make(map[string]persistence.ConfigParameter)
	startParams := configs[0].Parameters

	for name, val := range startParams {
		if isSharedParameter(configs[1:], name, val) {
			result[name] = val
		}
	}
	return result
}

func isSharedParameter(configs []extendedConfigDefinition, name string, val persistence.ConfigParameter) bool {
	for _, conf := range configs {
		paramVal := conf.Parameters[name]

		if !reflect.DeepEqual(val, paramVal) {
			return false
		}
	}
	return true
}

type propertyCheckResult struct {
	shareName bool
	foundName bool
	name      persistence.ConfigParameter

	shareTemplate bool
	foundTemplate bool
	template      string

	shareSkip bool
	foundSkip bool
	skip      interface{}
}

func testForSameProperties(configs []extendedConfigDefinition) propertyCheckResult {
	name := configs[0].Name
	templ := configs[0].Template
	skip := configs[0].Skip

	var (
		sameName,
		sameTemplate,
		sameSkip = true, true, true
	)

	for _, c := range configs {
		sameName = sameName && reflect.DeepEqual(name, c.Name)
		sameTemplate = sameTemplate && templ == c.Template
		sameSkip = sameSkip && (reflect.DeepEqual(skip, c.Skip) ||
			(skip == nil && c.Skip == false) ||
			(skip == false && c.Skip == nil))
	}

	if !sameName {
		name = nil
	}

	if !sameTemplate {
		templ = ""
	}

	if !sameSkip {
		skip = nil
	}

	return propertyCheckResult{
		shareName: sameName,
		foundName: name != nil || !sameName,
		name:      name,

		shareTemplate: sameTemplate,
		foundTemplate: templ != "" || !sameTemplate,
		template:      templ,

		shareSkip: sameSkip,
		foundSkip: skip != nil || !sameSkip,
		skip:      skip,
	}
}

type extendedConfigDefinition struct {
	persistence.ConfigDefinition
	group       string
	environment string
}

func toConfigDefinitions(context *serializerContext, configs []config.Config) ([]extendedConfigDefinition, []configTemplate, []error) {
	var errs []error
	result := make([]extendedConfigDefinition, 0, len(configs))

	var templates []configTemplate

	for _, c := range configs {
		definition, templ, convertErrs := toConfigDefinition(context, c)

		if len(convertErrs) > 0 {
			errs = append(errs, convertErrs...)
			continue
		}

		templates = append(templates, templ)

		result = append(result, extendedConfigDefinition{
			ConfigDefinition: definition,
			group:            c.Group,
			environment:      c.Environment,
		})
	}

	if len(errs) > 0 {
		return nil, nil, errs
	}

	return result, templates, nil
}

func toConfigDefinition(context *serializerContext, cfg config.Config) (persistence.ConfigDefinition, configTemplate, []error) {
	var errs []error
	detailedContext := detailedSerializerContext{
		serializerContext: context,
		environmentDetails: environmentDetails{
			group:       cfg.Group,
			environment: cfg.Environment,
		},
	}
	nameParam, err := parseNameParameter(&detailedContext, cfg)
	if err != nil {
		errs = append(errs, err)
	}

	params, convertErrs := convertParameters(&detailedContext, cfg.Parameters)

	errs = append(errs, convertErrs...)

	configTemplatePath, templ, err := extractTemplate(&detailedContext, cfg)

	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return persistence.ConfigDefinition{}, configTemplate{}, errs
	}

	return persistence.ConfigDefinition{
		Name:           nameParam,
		Parameters:     params,
		Template:       filepath.ToSlash(configTemplatePath),
		Skip:           cfg.Skip,
		OriginObjectId: cfg.OriginObjectId,
	}, templ, nil
}

func extractTemplate(context *detailedSerializerContext, cfg config.Config) (string, configTemplate, error) {
	var name, path string
	switch t := cfg.Template.(type) {
	case *template.InMemoryTemplate:
		if t.FilePath() != nil {
			path = *t.FilePath()
			n, err := filepath.Rel(context.configFolder, filepath.Clean(path))
			if err != nil {
				return "", configTemplate{}, newDetailedConfigWriterError(context.serializerContext, err)
			}
			name = n
		} else {
			name = prepareFileName(t.ID(), ".json")
			path = filepath.Join(context.configFolder, name)
		}
	default:
		return "", configTemplate{}, newDetailedConfigWriterError(context.serializerContext, fmt.Errorf("can not persist unexpected template type %q", t))
	}

	content, err := cfg.Template.Content()
	if err != nil {
		return "", configTemplate{}, newDetailedConfigWriterError(context.serializerContext, err)
	}

	return name, configTemplate{
		templatePath: path,
		content:      content,
	}, nil
}

func convertParameters(context *detailedSerializerContext, parameters config.Parameters) (map[string]persistence.ConfigParameter, []error) {
	var errs []error
	result := make(map[string]persistence.ConfigParameter)

	for name, param := range parameters {
		// ignore NameParameter and ScopeParameter as it is handled in a special way
		if name == config.NameParameter || name == config.ScopeParameter || name == config.InsertAfterParameter {
			continue
		}

		parsed, err := toParameterDefinition(context, name, param)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		result[name] = parsed
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return result, nil
}

func parseNameParameter(context *detailedSerializerContext, cfg config.Config) (persistence.ConfigParameter, error) {
	nameParam, found := cfg.Parameters[config.NameParameter]

	if !found {
		return nil, nil // not having a name is fine for some API types
	}

	return toParameterDefinition(context, config.NameParameter, nameParam)
}

func toParameterDefinition(context *detailedSerializerContext, parameterName string,
	param parameter.Parameter) (persistence.ConfigParameter, error) {

	if isValueParameter(param) {
		return toValueShorthandDefinition(context, parameterName, param)
	}

	serde, found := context.ParametersSerde[param.GetType()]

	if !found {
		return nil, fmtDetailedConfigWriterError(context.serializerContext,
			fmt.Errorf("%s:%s: no serde found for type `%s`", context.config, parameterName, param.GetType()))
	}

	result, err := serde.Serializer(newParameterSerializerContext(context, parameterName, param))

	if err != nil {
		return nil, err
	}

	result["type"] = param.GetType()

	return result, nil
}

func isValueParameter(param parameter.Parameter) bool {
	return param.GetType() == value.ValueParameterType
}

func toValueShorthandDefinition(context *detailedSerializerContext, parameterName string,
	param parameter.Parameter) (persistence.ConfigParameter, error) {
	if param.GetType() == value.ValueParameterType {
		valueParam, ok := param.(*value.ValueParameter)

		if !ok {
			return nil, fmtDetailedConfigWriterError(context.serializerContext,
				fmt.Errorf("%s:%s: parameter of type `%s` is no value param", context.config, parameterName, param.GetType()))
		}

		switch valueParam.Value.(type) {
		// strings can be shorthanded
		case string:
			return valueParam.Value, nil
		default:
			result, err := context.ParametersSerde[param.GetType()].Serializer(newParameterSerializerContext(context, parameterName, param))

			if err != nil {
				return nil, err
			}

			result["type"] = valueParam.GetType()

			return result, nil
		}
	}

	return nil, fmtDetailedConfigWriterError(context.serializerContext, fmt.Errorf("%s:%s: unknown special type `%s`", context.config, parameterName, param.GetType()))
}

type extendedCoordinate struct {
	coordinate.Coordinate
	extendedType string
}

func newExtendedCoordinateFromConfig(c config.Config) extendedCoordinate {
	switch t := c.Type.(type) {
	case config.DocumentType:
		return extendedCoordinate{
			Coordinate:   c.Coordinate,
			extendedType: fmt.Sprintf("%s-%s", c.Coordinate.Type, string(t.Kind)),
		}

	default:
		return extendedCoordinate{
			Coordinate:   c.Coordinate,
			extendedType: c.Coordinate.Type,
		}
	}
}

func groupConfigs(configs []config.Config) map[extendedCoordinate][]config.Config {
	result := make(map[extendedCoordinate][]config.Config)

	for _, c := range configs {
		eCoord := newExtendedCoordinateFromConfig(c)
		result[eCoord] = append(result[eCoord], c)
	}

	return result
}

func newParameterSerializerContext(context *detailedSerializerContext, name string,
	param parameter.Parameter) parameter.ParameterWriterContext {
	return parameter.ParameterWriterContext{
		Coordinate:    context.config,
		Group:         context.environmentDetails.group,
		Environment:   context.environmentDetails.environment,
		ParameterName: name,
		Parameter:     param,
	}
}

func newConfigWriterError(context *WriterContext, err error) configError.DetailedConfigWriterError {
	return configError.DetailedConfigWriterError{
		Path: filepath.Join(context.OutputFolder, context.ProjectFolder),
		Err:  err,
	}
}

func newDetailedConfigWriterError(context *serializerContext, err error) configError.DetailedConfigWriterError {
	return configError.DetailedConfigWriterError{
		Path:     context.configFolder,
		Location: context.config,
		Err:      err,
	}
}

func fmtDetailedConfigWriterError(context *serializerContext, err error) configError.DetailedConfigWriterError {

	return configError.DetailedConfigWriterError{
		Path:     context.configFolder,
		Location: context.config,
		Err:      err,
	}
}

// prepareFileName makes sure that a given file name meets all requirements like no forbidden characters
// and max file name length. It takes the name (without file extension) and the file extension (with the separating ".", e.g. ".json")
// and returns the filename combined with the file extension
func prepareFileName(name string, fileExtension string) string {

	const reservedForUniqueCounter = 2
	maxFileNameLen := environment.GetEnvValueInt(environment.MaxFilenameLenKey)
	sanitizedName := mystrings.Sanitize(name)

	maxLen := maxFileNameLen
	if len(fileExtension)+reservedForUniqueCounter <= maxFileNameLen {
		maxLen = maxFileNameLen - len(fileExtension) - reservedForUniqueCounter
	}

	runes := []rune(sanitizedName)
	if len(runes) > maxLen {
		sanitizedName = string(runes[:maxLen])
	}

	finishedName := getUniqueFileName(sanitizedName) + fileExtension

	if len(finishedName) > maxFileNameLen {
		panic("cannot use file name " + finishedName + " as it is too long")
	}

	return finishedName
}

var fileNameClashes = make(map[string]int)

func getUniqueFileName(name string) string {
	if _, ok := fileNameClashes[name]; ok {
		fileNameClashes[name]++
		return fmt.Sprintf("%s%d", name, fileNameClashes[name])
	}
	fileNameClashes[name] = 0
	return name
}
