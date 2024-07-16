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

//go:build unit

package converter

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	compoundParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/converter/v1environment"
	"reflect"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	listParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/list"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	projectV1 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v1"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

const simpleParameterName = "randomValue"
const referenceParameterName = "managementZoneId"
const referenceParameterLongName = "managementZoneIdLong"
const referenceToCurProjName = "managementZoneId2"
const listParameterName = "locations"

func TestConvertParameters(t *testing.T) {
	environmentName := "test"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	referenceToAnotherProjParameterValue := "/projectB/management-zone/zone.id"
	referenceToAnotherProjParameterLongValue := "some_path/projectB/management-zone/zone.id"
	referenceToCurrentProjParameterValue := "management-zone/zone.id"
	listParameterValue := `"GEOLOCATION-41","GEOLOCATION-42","GEOLOCATION-43"`
	envParameterName := "url"
	envParameterValue := "{{ .Env.SOME_ENV_VAR }}"
	compoundEnvParameterName := "nickname"
	compoundEnvParameterValue := "something {{ .Env.TITLE }} - something {{ .Env.NICKNAME }} {{ .Env.SURNAME }} - something"

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		KnownListParameterIds: map[string]struct{}{listParameterName: {}},
		V1Apis: api.APIs{
			"alerting-profile": testApi,
			"management-zone":  api.API{ID: "management-zone", URLPath: "/api/path"},
		},
		ProjectId: "projectA",
	}

	properties := map[string]map[string]string{
		configId: {
			"name":                     configName,
			simpleParameterName:        simpleParameterValue,
			referenceParameterName:     referenceToAnotherProjParameterValue,
			referenceParameterLongName: referenceToAnotherProjParameterLongValue,
			referenceToCurProjName:     referenceToCurrentProjParameterValue,
			listParameterName:          listParameterValue,
			envParameterName:           envParameterValue,
			compoundEnvParameterName:   compoundEnvParameterValue,
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	parameters, skip, errors := convertParameters(convertContext, environment, testConfig)

	assert.Nil(t, errors)
	assert.Equal(t, 11, len(parameters))
	assert.Nil(t, skip, "should be empty")

	nameParameter, found := parameters["name"]

	assert.Equal(t, true, found)
	assert.Equal(t, configName, nameParameter.(*valueParam.ValueParameter).Value)

	simpleParameter, found := parameters[simpleParameterName]

	assert.Equal(t, true, found)
	assert.Equal(t, simpleParameterValue, simpleParameter.(*valueParam.ValueParameter).Value)

	assert.Equal(t, refParam.New("projectB", "management-zone", "zone", "id"), parameters[referenceParameterName].(*refParam.ReferenceParameter))
	assert.Equal(t, refParam.New("some_path.projectB", "management-zone", "zone", "id"), parameters[referenceParameterLongName].(*refParam.ReferenceParameter))
	assert.Equal(t, refParam.New("projectA", "management-zone", "zone", "id"), parameters[referenceToCurProjName].(*refParam.ReferenceParameter))

	listParameter, found := parameters[listParameterName]
	assert.Equal(t, true, found)
	assert.Equal(t, []valueParam.ValueParameter{{"GEOLOCATION-41"}, {"GEOLOCATION-42"}, {"GEOLOCATION-43"}}, listParameter.(*listParam.ListParameter).Values)

	envParameter, found := parameters[envParameterName]
	assert.Equal(t, true, found)
	assert.Equal(t, "SOME_ENV_VAR", envParameter.(*envParam.EnvironmentVariableParameter).Name)
	//assert.Len(t, envParameter.(*compoundParam.CompoundParameter).GetReferences(), 1)
	//assert.Equal(t, "__ENV_SOME_ENV_VAR__", envParameter.(*compoundParam.CompoundParameter).GetReferences()[0].Property)

	compound := parameters["nickname"]

	expectedCompoundParam, _ := compoundParam.New("nickname", "something {{ .__ENV_TITLE__ }} - something {{ .__ENV_NICKNAME__ }} {{ .__ENV_SURNAME__ }} - something", []parameter.ParameterReference{
		{
			Config: coordinate.Coordinate{
				Project:  "test-project",
				Type:     "alerting-profile",
				ConfigId: "alerting-profile-1",
			},
			Property: "__ENV_TITLE__",
		},
		{
			Config: coordinate.Coordinate{
				Project:  "test-project",
				Type:     "alerting-profile",
				ConfigId: "alerting-profile-1",
			},
			Property: "__ENV_NICKNAME__",
		},
		{
			Config: coordinate.Coordinate{
				Project:  "test-project",
				Type:     "alerting-profile",
				ConfigId: "alerting-profile-1",
			},
			Property: "__ENV_SURNAME__",
		},
	})
	assert.Equal(t, expectedCompoundParam, compound)
}

func TestConvertConvertRemovesEscapeCharsFromParameters(t *testing.T) {
	environmentName := "test"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	simpleId := "simpleEscapeParam"
	simpleEsc := `\"hello\"`
	stringId := "stringEscapeParam"
	stringEsc := "\\\"hello\\\""
	severalId := "severalEscapeParam"
	severalEsc := "\\\"one line\\ntwo line\\\""

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		KnownListParameterIds: map[string]struct{}{listParameterName: {}},
		V1Apis: api.APIs{
			"alerting-profile": testApi,
			"management-zone":  api.API{ID: "management-zone", URLPath: "/api/path"},
		},
		ProjectId: "projectA",
	}

	properties := map[string]map[string]string{
		configId: {
			"name":    configName,
			simpleId:  simpleEsc,
			stringId:  stringEsc,
			severalId: severalEsc,
		},
	}

	content := fmt.Sprintf(`{ "a": "{.%s}", "b": "{.%s}", "b": "{.%s}"`, simpleId, stringId, severalId)
	tmpl, err := template.NewTemplateFromString("test/test-configV1.json", content)

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		tmpl, properties, testApi)

	assert.NoError(t, err)

	parameters, skip, errors := convertParameters(convertContext, environment, testConfig)

	assert.Nil(t, errors)
	assert.Equal(t, 4, len(parameters))
	assert.Nil(t, skip, "should be empty")

	nameParameter, found := parameters["name"]

	assert.Equal(t, true, found)
	assert.Equal(t, configName, nameParameter.(*valueParam.ValueParameter).Value)

	simpleParameter, found := parameters[simpleId]
	assert.Equal(t, true, found)
	assert.Equal(t, `"hello"`, simpleParameter.(*valueParam.ValueParameter).Value)

	stringParameter, found := parameters[stringId]
	assert.Equal(t, true, found)
	assert.Equal(t, `"hello"`, stringParameter.(*valueParam.ValueParameter).Value)

	severalParameter, found := parameters[severalId]
	assert.Equal(t, true, found)
	assert.Equal(t, `"one line
two line"`, severalParameter.(*valueParam.ValueParameter).Value)
}

func TestParseSkipDeploymentParameter(t *testing.T) {
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: "projectA",
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":                                  configName,
			projectV1.SkipConfigDeploymentParameter: "true",
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	testCases := []struct {
		shouldFail    bool
		testValue     string
		expectedValue parameter.Parameter
	}{
		{
			shouldFail:    false,
			testValue:     "true",
			expectedValue: valueParam.New(true),
		},
		{
			shouldFail:    false,
			testValue:     "TRue",
			expectedValue: valueParam.New(true),
		},
		{
			shouldFail:    false,
			testValue:     "false",
			expectedValue: valueParam.New(false),
		},
		{
			shouldFail:    false,
			testValue:     "FaLse",
			expectedValue: valueParam.New(false),
		},
		{
			shouldFail: true,
			testValue:  "",
		},
		{
			shouldFail: true,
			testValue:  "tru",
		},
		{
			shouldFail: true,
			testValue:  "aaaaaa",
		},
		{
			shouldFail:    false,
			testValue:     "{{          .Env.TEST_VAR           }}",
			expectedValue: envParam.New("TEST_VAR"),
		},
		{
			shouldFail:    false,
			testValue:     "{{.Env.TEST_VAR}}",
			expectedValue: envParam.New("TEST_VAR"),
		},
		{
			shouldFail: true,
			testValue:  "{{ .TEST_VAR }}",
		},
	}

	for _, c := range testCases {
		skip, err := parseSkipDeploymentParameter(convertContext, testConfig, c.testValue)

		if c.shouldFail {
			assert.NotNilf(t, err, "there should be an error for `%s`", c.testValue)
		} else {
			assert.Nilf(t, err, "there should be no error for `%s`", c.testValue)
			assert.Equal(t, c.expectedValue, skip, "should be `%t` for `%s`", c.expectedValue, c.testValue)
		}
	}
}

func TestLoadPropertiesForEnvironment(t *testing.T) {
	environmentName := "dev"
	groupName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: groupName,
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    "wrong",
			referenceParameterName: "wrong",
		},
		configId + "." + "unknown": {
			"name": configName,
		},
		configId + "." + groupName: {
			simpleParameterName: simpleParameterValue,
		},
		configId + "." + environmentName: {
			referenceParameterName: referenceParameterValue,
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	envProperties := loadPropertiesForEnvironment(environment, testConfig)

	assert.Equal(t, configName, envProperties["name"])
	assert.Equal(t, simpleParameterValue, envProperties[simpleParameterName])
	assert.Equal(t, referenceParameterValue, envProperties[referenceParameterName])
}

func TestConvertConfig(t *testing.T) {
	projectId := "projectA"
	environmentName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/management-zone/zone.id"
	envVarName := "TEST_VAR"

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}
	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		V1Apis:    api.NewV1APIs(),
		ProjectId: "projectA",
	}

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
			"scope":                "value",
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, testConfig)

	assert.Equal(t, 0, len(errors), "errors: %s", errors)
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, testApi.ID, convertedConfig.Coordinate.Type)
	assert.Equal(t, configId, convertedConfig.Coordinate.ConfigId)
	assert.Equal(t, environmentName, convertedConfig.Environment)

	references := convertedConfig.References()

	assert.Equal(t, 1, len(references))
	assert.Equal(t, "projectB", references[0].Project)
	assert.Equal(t, "management-zone", references[0].Type)
	assert.Equal(t, "zone", references[0].ConfigId)

	assert.Equal(t, 5, len(convertedConfig.Parameters))
	assert.Equal(t, configName, convertedConfig.Parameters["name"].(*valueParam.ValueParameter).Value)
	assert.Equal(t, simpleParameterValue, convertedConfig.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)
	assert.Equal(t, envVarName, convertedConfig.Parameters[transformEnvironmentToParamName(envVarName)].(*envParam.EnvironmentVariableParameter).Name)
	assert.Equal(t, "value", convertedConfig.Parameters["scope1"].(*valueParam.ValueParameter).Value)
	assert.Equal(t, refParam.New("projectB", "management-zone", "zone", "id"), convertedConfig.Parameters[referenceParameterName].(*refParam.ReferenceParameter))
}

func TestConvertDeprecatedConfigToLatest(t *testing.T) {
	projectId := "projectA"
	environmentName := "development"
	configId := "application-1"
	configName := "Application 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/application/another-app.id"
	envVarName := "TEST_VAR"

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	deprecatedApi := api.API{ID: "application", URLPath: "/api/configV1/v1/application/web", DeprecatedBy: "application-web"}

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		V1Apis:    api.APIs{"application": deprecatedApi},
		ProjectId: "projectA",
	}

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, deprecatedApi)

	assert.NoError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, testConfig)

	assert.Equal(t, 0, len(errors), "errors: %s", errors)
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, deprecatedApi.DeprecatedBy, convertedConfig.Coordinate.Type)
	assert.Equal(t, configId, convertedConfig.Coordinate.ConfigId)
	assert.Equal(t, environmentName, convertedConfig.Environment)

	references := convertedConfig.References()

	assert.Equal(t, 1, len(references))
	assert.Equal(t, "projectB", references[0].Project)
	assert.Equal(t, "application-web", references[0].Type, "expected deprecated API in reference to be replaced as well")
	assert.Equal(t, "another-app", references[0].ConfigId)

	assert.Equal(t, 4, len(convertedConfig.Parameters))
	assert.Equal(t, configName, convertedConfig.Parameters["name"].(*valueParam.ValueParameter).Value)
	assert.Equal(t, simpleParameterValue, convertedConfig.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)
	assert.Equal(t, envVarName,
		convertedConfig.Parameters[transformEnvironmentToParamName(envVarName)].(*envParam.EnvironmentVariableParameter).Name)
}

func TestConvertConfigWithEnvNameCollisionShouldFail(t *testing.T) {
	environmentName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	envVarName := "COLLISION"
	simpleParameterName := transformEnvironmentToParamName(envVarName)
	simpleParameterValue := "hello"

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		ProjectId: "projectA",
	}

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":              configName,
			simpleParameterName: simpleParameterValue,
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	_, errors := convertConfig(convertContext, environment, testConfig)

	assert.Greater(t, len(errors), 0, "expected errors, but got none")
}

func TestConvertSkippedConfig(t *testing.T) {
	projectId := "projectA"
	environmentName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: "projectA",
	}

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		URL:   createSimpleUrlDefinition(),
		Group: "",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "token"},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":                                  configName,
			projectV1.SkipConfigDeploymentParameter: "true",
		},
	}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, testConfig)

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, testApi.ID, convertedConfig.Coordinate.Type)
	assert.Equal(t, configId, convertedConfig.Coordinate.ConfigId)
	assert.Equal(t, environmentName, convertedConfig.Environment)
	assert.Equal(t, valueParam.New(true), convertedConfig.SkipForConversion)
}

func TestConvertConfigs(t *testing.T) {
	projectId := "projectA"
	environmentName := "dev"
	environmentGroup := "development"
	environmentName2 := "sprint"
	environmentGroup2 := "hardening"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	simpleParameterValue2 := "world"
	referenceParameterValue := "/projectB/management-zone/zone.id"
	listParameterValue := `"GEOLOCATION-41","GEOLOCATION-42","GEOLOCATION-43"`
	listParameterValue2 := `"james.t.kirk@dynatrace.com"`
	envVariableName := "ENV_VAR"

	environments := map[string]manifest.EnvironmentDefinition{
		environmentName: manifest.EnvironmentDefinition{
			Name:  environmentName,
			URL:   createSimpleUrlDefinition(),
			Group: environmentGroup,
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: "token"},
			},
		},
		environmentName2: manifest.EnvironmentDefinition{
			Name:  environmentName2,
			URL:   createSimpleUrlDefinition(),
			Group: environmentGroup2,
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: "token"},
			},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":                 fmt.Sprintf("%s {{ .Env.%s }}", configName, envVariableName),
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
			listParameterName:      listParameterValue,
		},
		configId + "." + environmentGroup2: {
			simpleParameterName: simpleParameterValue2,
			listParameterName:   listParameterValue2,
		},
	}

	fs, template := setupFsWithFullTestTemplate(t, simpleParameterName, referenceParameterName, listParameterName, envVariableName)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		V1Apis:    api.NewV1APIs(),
		ProjectId: projectId,
	}

	convertedConfigs, errors := convertConfigs(convertContext, environments, []*projectV1.Config{testConfig})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 2, len(convertedConfigs))

	apiConfigs := convertedConfigs[environmentName]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[testApi.ID]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, configId, c.Coordinate.ConfigId)
	assert.Equal(t, 5, len(c.Parameters))

	// assert value param is converted as expected
	assert.Equal(t, simpleParameterValue, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	// assert value param is converted as expected
	assert.Equal(t, coordinate.Coordinate{
		Project:  "projectB",
		Type:     "management-zone",
		ConfigId: "zone",
	}, c.Parameters[referenceParameterName].(*refParam.ReferenceParameter).Config)
	assert.Equal(t, "id", c.Parameters[referenceParameterName].(*refParam.ReferenceParameter).Property)

	// assert list param is converted as expected
	assert.Equal(t, []valueParam.ValueParameter{{"GEOLOCATION-41"}, {"GEOLOCATION-42"}, {"GEOLOCATION-43"}}, c.Parameters[listParameterName].(*listParam.ListParameter).Values)

	transformedEnvVarName := transformEnvironmentToParamName(envVariableName)
	// assert env reference in template has created correct env parameter
	assert.Equal(t, envVariableName, c.Parameters[transformedEnvVarName].(*envParam.EnvironmentVariableParameter).Name)

	nameCompound, err := compoundParam.New(config.NameParameter, fmt.Sprintf("%s {{ .%s }}", configName, transformedEnvVarName), []parameter.ParameterReference{
		{
			Config: coordinate.Coordinate{
				Project:  "test-project",
				Type:     "alerting-profile",
				ConfigId: configId,
			},
			Property: transformedEnvVarName,
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, nameCompound, c.Parameters[config.NameParameter].(*compoundParam.CompoundParameter))

	apiConfigs = convertedConfigs[environmentName2]
	assert.Equal(t, 1, len(apiConfigs))

	configs = apiConfigs[testApi.ID]
	assert.Equal(t, 1, len(configs))

	c = configs[0]
	assert.Equal(t, configId, c.Coordinate.ConfigId)
	assert.Equal(t, 5, len(c.Parameters))

	// assert override simple param is converted as expected
	assert.Equal(t, simpleParameterValue2, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	// assert override list param is converted as expected
	// assert list param is converted as expected
	assert.Equal(t, []valueParam.ValueParameter{{"james.t.kirk@dynatrace.com"}}, c.Parameters[listParameterName].(*listParam.ListParameter).Values)
}

func TestConvertWithMissingName(t *testing.T) {
	environments := map[string]manifest.EnvironmentDefinition{
		"dev": manifest.EnvironmentDefinition{
			Name:  "dev",
			URL:   createSimpleUrlDefinition(),
			Group: "development",
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: "token"},
			},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		"alerting-profile-1": {},
	}

	fs := afero.NewMemMapFs()
	err := fs.Mkdir("test", 0644)
	assert.NoError(t, err)

	templ, err := template.NewTemplateFromString("test/test-configV1.json", "")
	assert.NoError(t, err)

	err = afero.WriteFile(fs, "test/test-configV1.json", []byte(""), 0644)
	assert.NoError(t, err)

	testConfig := projectV1.NewConfigWithTemplate("alerting-profile-1", "test-project", "test/test-configV1.json", templ, properties, testApi)

	convertContext := &configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		V1Apis:    api.NewV1APIs(),
		ProjectId: "projectA",
	}

	convertedConfigs, errors := convertConfigs(convertContext, environments, []*projectV1.Config{testConfig})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 1, len(convertedConfigs))

	apiConfigs := convertedConfigs[("dev")]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[testApi.ID]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, "alerting-profile-1", c.Coordinate.ConfigId)
	assert.Equal(t, 1, len(c.Parameters))

	// assert value param is converted as expected
	assert.Equal(t, "alerting-profile-1 - monaco-conversion created name", c.Parameters[config.NameParameter].(*valueParam.ValueParameter).Value)
}

func TestConvertProjects(t *testing.T) {
	projectId := "projectA"
	environmentName := "dev"
	environmentGroup := "development"
	environmentName2 := "sprint"
	environmentGroup2 := "hardening"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	simpleParameterValue2 := "world"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	convertContext := &ConverterContext{
		Fs: setupDummyFs(t),
	}

	environments := map[string]manifest.EnvironmentDefinition{

		environmentName: manifest.EnvironmentDefinition{
			Name:  environmentName,
			URL:   createSimpleUrlDefinition(),
			Group: environmentGroup,
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: "token"},
			},
		},
		environmentName2: manifest.EnvironmentDefinition{
			Name:  environmentName2,
			URL:   createSimpleUrlDefinition(),
			Group: environmentGroup2,
			Auth: manifest.Auth{
				Token: manifest.AuthSecret{Name: "token"},
			},
		},
	}

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
		},
		configId + "." + environmentGroup2: {
			simpleParameterName: simpleParameterValue2,
		},
	}

	template := generateDummyTemplate(t)

	testConfig := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	project := &projectV1.ProjectImpl{
		Id:      projectId,
		Configs: []*projectV1.Config{testConfig},
	}

	projectDefinitions, convertedProjects, errors := convertProjects(convertContext, environments, []projectV1.Project{project})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 1, len(projectDefinitions))
	assert.Equal(t, 1, len(convertedProjects))

	projectDefinition := projectDefinitions[projectId]
	convertedProject := convertedProjects[0]

	assert.Equal(t, projectId, projectDefinition.Name)
	assert.Equal(t, projectId, projectDefinition.Path)
	assert.Nil(t, convertedProject.Dependencies, "Dependencies should not be resolved")

	convertedConfigs := convertedProject.Configs

	assert.Equal(t, 2, len(convertedConfigs))

	apiConfigs := convertedConfigs[environmentName]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[testApi.ID]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, configId, c.Coordinate.ConfigId)
	assert.Equal(t, 3, len(c.Parameters))
	assert.Equal(t, simpleParameterValue, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	apiConfigs = convertedConfigs[environmentName2]
	assert.Equal(t, 1, len(apiConfigs))

	configs = apiConfigs[testApi.ID]
	assert.Equal(t, 1, len(configs))

	c = configs[0]
	assert.Equal(t, configId, c.Coordinate.ConfigId)
	assert.Equal(t, 3, len(c.Parameters))
	assert.Equal(t, simpleParameterValue2, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)
}

func TestConvertTemplate_ConvertsEnvReferences(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "test.json", []byte(`{
		"test": "{{.Env.HELLO}}",
		"test1": "{{ .Env.HELLO }}",
		"test2": "{{  .Env.HELLO_WORLD}} {{ .Env.NAME }}",
		"test3": "{{  .Env.HELLO_WORLD}} {{ .Env.HE     }}",
	}`), 0644)

	assert.NoError(t, err)

	templ, envParams, _, errs := convertTemplate(&configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Len(t, errs, 0)
	assert.NotNil(t, templ)

	for _, env := range []string{
		"HELLO",
		"HELLO_WORLD",
		"NAME",
		"HE",
	} {
		paramName := transformEnvironmentToParamName(env)
		param := envParams[paramName]

		assert.Containsf(t, envParams, paramName, "should contain `%s`", paramName)
		assert.NotNilf(t, param, "param `%s` should be not nil", paramName)

		assert.IsTypef(t, &envParam.EnvironmentVariableParameter{}, param, "param `%s` should be an environment variable", paramName)
		e := param.(*envParam.EnvironmentVariableParameter)
		assert.Equalf(t, false, e.HasDefaultValue, "param `%v` should have no default value", e)
		assert.Equal(t, env, e.Name)
	}
}

func TestConvertTemplate_ConvertsListVariables(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "test.json", []byte(`{
		"test": [ {{.list}} ],
		"test1": [
				{{ .list1   }}
		],
	}`), 0644)

	assert.NoError(t, err)

	templ, _, listParamIds, errs := convertTemplate(&configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Len(t, errs, 0)
	assert.NotNil(t, templ)

	assert.Equal(t, len(listParamIds), 2, " expected to list param ids to be found in template")
	assert.Contains(t, listParamIds, "list")
	assert.Contains(t, listParamIds, "list1")
	gotContent, err := templ.Content()
	assert.NoError(t, err)
	assert.Equal(t, gotContent, `{
		"test": {{ .list }},
		"test1": {{ .list1 }},
	}`)
}

func TestConvertTemplate(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "test.json", []byte(`{
		"envKey": "{{.Env.ENV_VALUE}}",
		"listKey": [
				{{ .list_value   }}
		],
		"key": "value"
	}`), 0644)

	assert.NoError(t, err)

	templ, envParams, listParamIds, errs := convertTemplate(&configConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Len(t, errs, 0)
	assert.NotNil(t, templ)

	// check list parameter
	assert.Equal(t, len(listParamIds), 1, " expected to list param ids to be found in template")
	assert.Contains(t, listParamIds, "list_value")

	// check env parameter
	paramName := transformEnvironmentToParamName("ENV_VALUE")
	assert.Contains(t, envParams, paramName)
	param := envParams[paramName]
	assert.NotNil(t, param)

	assert.IsTypef(t, &envParam.EnvironmentVariableParameter{}, param, "EnvParam `%s` should be an environment variable", paramName)
	p := param.(*envParam.EnvironmentVariableParameter)
	assert.Equal(t, false, p.HasDefaultValue)
	assert.Equal(t, "ENV_VALUE", p.Name)

	// check converted template
	gotContent, err := templ.Content()
	assert.NoError(t, err)
	assert.Equal(t, gotContent, `{
		"envKey": "{{ .__ENV_ENV_VALUE__ }}",
		"listKey": {{ .list_value }},
		"key": "value"
	}`)
}

func TestConvertListsInTemplate(t *testing.T) {
	input := `{
		"test": [ {{.list}} ],
		"test1": [
				{{ .list1   }}
		],
	}`
	expected := `{
		"test": {{ .list }},
		"test1": {{ .list1 }},
	}`
	result, paramIds, errs := convertListsInTemplate(input, "does/not/matter")
	assert.Len(t, errs, 0)
	assert.Len(t, paramIds, 2)

	assert.Contains(t, paramIds, "list")
	assert.Contains(t, paramIds, "list1")

	assert.Equal(t, result, expected)
}

func TestAdjustProjectId(t *testing.T) {
	id := adjustProjectId(`test\project/name`)

	assert.Equal(t, `test.project.name`, id)
}

func Test_parseListStringToValueSlice(t *testing.T) {
	tests := []struct {
		inputString string
		want        []valueParam.ValueParameter
		wantErr     bool
	}{
		{
			`"a", "b", "c"`,
			[]valueParam.ValueParameter{{"a"}, {"b"}, {"c"}},
			false,
		},
		{
			`  " a " , " b "`,
			[]valueParam.ValueParameter{{" a "}, {" b "}},
			false,
		},
		{
			`  "e@mail.com" , "first.last@domain.com"  `,
			[]valueParam.ValueParameter{{"e@mail.com"}, {"first.last@domain.com"}},
			false,
		},
		{
			`  " a " , " b "   , `,
			[]valueParam.ValueParameter{{" a "}, {" b "}},
			false,
		},
		{
			`"a"`,
			[]valueParam.ValueParameter{{"a"}},
			false,
		},
		{
			`"e@mail.com"`,
			[]valueParam.ValueParameter{{"e@mail.com"}},
			false,
		},
		{
			``,
			[]valueParam.ValueParameter{},
			true,
		},
		{
			`"inval,id`,
			[]valueParam.ValueParameter{},
			true,
		},
		{
			`"",`,
			[]valueParam.ValueParameter{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.inputString, func(t *testing.T) {
			got, err := parseListStringToValueSlice(tt.inputString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseListStringToSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseListStringToSlice() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseReference(t *testing.T) {
	tests := []struct {
		name               string
		givenParameterName string
		givenReference     string
		givenKnownApis     api.APIs
		want               *refParam.ReferenceParameter
		wantErr            bool
	}{
		{
			"parses relative reference",
			"test-param",
			"some-project/alerting-profile/some-configV1.id",
			api.NewV1APIs(),
			refParam.New("some-project", "alerting-profile", "some-configV1", "id"),
			false,
		},
		{
			"parses absolute reference",
			"test-param",
			"/some-project/alerting-profile/some-configV1.id",
			api.NewV1APIs(),
			refParam.New("some-project", "alerting-profile", "some-configV1", "id"),
			false,
		},
		{
			"returns error for invalid reference",
			"test-param",
			"/management-zone/zone.id",
			api.NewV1APIs(),
			refParam.New("test-project", "management-zone", "zone", "id"),
			false,
		},
		{
			"returns error for non-reference",
			"test-param",
			"not-a-reference",
			api.NewV1APIs(),
			nil,
			true,
		},
		{
			"returns error for unknown api reference",
			"test-param",
			"/some-project/alerting-profile/some-configV1.id",
			api.APIs{}, //no APIs known
			nil,
			true,
		},
		{
			"replaces deprecated APIs",
			"test-param",
			"/some-project/deprecated-api/some-configV1.some-property",
			api.APIs{
				"deprecated-api": api.API{ID: "deprecated-api", URLPath: "/api/path", DeprecatedBy: "new-api"},
			},
			refParam.New("some-project", "new-api", "some-configV1", "some-property"),
			false,
		},
		{
			"resolve reference with longer path at the start",
			"test-param",
			"/movies/science fiction/the-hitchhikers-guide-to-the-galaxy/management-zone/zone-multiproject.id",
			api.NewV1APIs(),
			refParam.New("movies.science fiction.the-hitchhikers-guide-to-the-galaxy", "management-zone", "zone-multiproject", "id"),
			false,
		},
		{
			"resolve reference within the same config",
			"test-param",
			"zone-multiproject.id",
			api.NewV1APIs(),
			refParam.New("test-project", "alerting-profile", "zone-multiproject", "id"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := generateDummyConfig(t)

			testContext := &configConvertContext{
				ConverterContext: &ConverterContext{
					Fs: setupDummyFs(t),
				},
				V1Apis:    tt.givenKnownApis,
				ProjectId: "test-project",
			}

			got, err := parseReference(testContext, testConfig, tt.givenParameterName, tt.givenReference)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseReference() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertReservedParameters(t *testing.T) {

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{
			name:     "scope is replaced",
			template: "{{ .scope }}",
			want:     "{{ .scope1 }}",
		},
		{
			name:     "name is not replaced",
			template: "{{ .name }}",
			want:     "{{ .name }}",
		},
		{
			name:     "generic var is not replaced",
			template: "{{ .scopeSomething }}",
			want:     "{{ .scopeSomething }}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertReservedParameters(tt.template); got != tt.want {
				t.Errorf("convertReservedParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEnvironmentDefinitionFromV1(t *testing.T) {
	type args struct {
		env   *v1environment.EnvironmentV1
		group string
	}
	tests := []struct {
		name string
		args args
		want manifest.EnvironmentDefinition
	}{
		{
			"simple v1 environment is converted",
			args{
				v1environment.NewEnvironmentV1("test", "name", "group", "http://google.com", "NAME"),
				"group",
			},
			createValueEnvironmentDefinition(),
		},
		{
			"v1 environment with env var is converted",
			args{
				v1environment.NewEnvironmentV1("test", "name", "group", "{{ .Env.ENV_VAR }}", "NAME"),
				"group",
			},
			createEnvEnvironmentDefinition(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (manifest.EnvironmentDefinition{
				Name:  tt.args.env.GetId(),
				URL:   newUrlDefinitionFromV1(tt.args.env),
				Group: tt.args.group,
				Auth: manifest.Auth{
					Token: manifest.AuthSecret{Name: tt.args.env.GetTokenName()},
				},
			}); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEnvironmentDefinitionFromV1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertToParameters(t *testing.T) {
	type args struct {
		envReference string
	}
	tests := []struct {
		name    string
		args    args
		want    []envParam.EnvironmentVariableParameter
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "convert to parameters - empty input - returns no params",
			args: args{
				envReference: "",
			},
			want:    []envParam.EnvironmentVariableParameter{},
			wantErr: assert.NoError,
		},
		{
			name: "convert to parameters - single env var - returns param",
			args: args{
				envReference: "{{ .Env.VAR1 }}",
			},
			want: []envParam.EnvironmentVariableParameter{
				*envParam.New("VAR1"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "convert to parameters - multiple env vars - returns params",
			args: args{
				envReference: "{{ .Env.VAR1 }} - {{ .Env.VAR2 }}",
			},
			want: []envParam.EnvironmentVariableParameter{
				*envParam.New("VAR1"),
				*envParam.New("VAR2"),
			},
			wantErr: assert.NoError,
		},
		{
			name: "convert to parameters - invalid input - returns error",
			args: args{
				envReference: "{{",
			},
			want:    []envParam.EnvironmentVariableParameter{},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEnvParameters(tt.args.envReference)
			assert.Equalf(t, tt.want, got, "extractEnvParameters(%v)", tt.args.envReference)
		})
	}
}

func setupDummyFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)

	assert.NoError(t, err)

	err = afero.WriteFile(fs, "test/test-configV1.json", []byte(`{}`), 0644)

	assert.NoError(t, err)

	return fs
}

func setupDummyFsWithEnvVariableInTemplate(t *testing.T, envVarName string) afero.Fs {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)

	assert.NoError(t, err)

	err = afero.WriteFile(fs, "test/test-configV1.json", []byte(fmt.Sprintf(`{"test": "{{.Env.%s}}"}`, envVarName)), 0644)

	assert.NoError(t, err)

	return fs
}

func setupFsWithFullTestTemplate(t *testing.T, simpleVar, refVar, listVar, envVar string) (afero.Fs, template.Template) {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)
	assert.NoError(t, err)

	templateContent := fmt.Sprintf(`{ "simple": "{{ .%s }}", "reference": "{{ .%s }}", "list": [ {{ .%s }} ], "env": "{{ .Env.%s }}" }`, simpleVar, refVar, listVar, envVar)

	template, err := template.NewTemplateFromString("test/test-configV1.json", templateContent)
	assert.NoError(t, err)

	err = afero.WriteFile(fs, "test/test-configV1.json", []byte(templateContent), 0644)
	assert.NoError(t, err)

	return fs, template
}

func generateDummyTemplate(t *testing.T) template.Template {
	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	return template
}

func generateDummyConfig(t *testing.T) *projectV1.Config {
	var configId = "alerting-profile-1"

	testApi := api.API{ID: "alerting-profile", URLPath: "/api/configV1/v1/alertingProfiles"}

	properties := map[string]map[string]string{}

	template, err := template.NewTemplateFromString("test/test-configV1.json", "{}")

	assert.NoError(t, err)

	conf := projectV1.NewConfigWithTemplate(configId, "test-project", "test/test-configV1.json",
		template, properties, testApi)

	assert.NoError(t, err)

	return conf
}
func createSimpleUrlDefinition() manifest.URLDefinition {
	return manifest.URLDefinition{
		Type:  manifest.ValueURLType,
		Value: "test.env",
	}
}

func createEnvEnvironmentDefinition() manifest.EnvironmentDefinition {
	return manifest.EnvironmentDefinition{
		Name: "test",
		URL: manifest.URLDefinition{
			Type: manifest.EnvironmentURLType,
			Name: "ENV_VAR",
		},
		Group: "group",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "NAME"},
		},
	}
}

func createValueEnvironmentDefinition() manifest.EnvironmentDefinition {
	return manifest.EnvironmentDefinition{
		Name: "test",
		URL: manifest.URLDefinition{
			Type:  manifest.ValueURLType,
			Value: "http://google.com",
		},
		Group: "group",
		Auth: manifest.Auth{
			Token: manifest.AuthSecret{Name: "NAME"},
		},
	}
}

func Test_removeEscapeChars(t *testing.T) {
	tests := []struct {
		given string
		want  string
	}{
		{
			`\"hello\"`,
			`"hello"`,
		},
		{
			"\\\"hello\\\"",
			`"hello"`,
		},
		{
			`\"hello slash (\) world!\"`,
			`"hello slash (\) world!"`,
		},
		{
			"\\\"one line\\ntwo line\\\"",
			`"one line
two line"`,
		},
		{
			"\\\"one line\\n\\rtwo line\\\"",
			"\"one line\n\rtwo line\"",
		},

		{
			"\\\"no tab\\tone tab\\\"",
			"\"no tab\tone tab\"",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q->%q", tt.given, tt.want), func(t *testing.T) {
			assert.Equal(t, tt.want, removeEscapeChars(tt.given))
		})
	}
}
