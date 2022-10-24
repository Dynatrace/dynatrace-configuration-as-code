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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	listParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/list"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	projectv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v1"
	"reflect"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	configv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	envParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const simpleParameterName = "randomValue"
const referenceParameterName = "managementZoneId"
const listParameterName = "locations"

func TestConvertParameters(t *testing.T) {
	environmentName := "test"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/management-zone/zone.id"
	listParameterValue := `"GEOLOCATION-41","GEOLOCATION-42","GEOLOCATION-43"`
	envParameterName := "url"
	envParameterValue := " {{ .Env.SOME_ENV_VAR }} "

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		KnownListParameterIds: map[string]struct{}{listParameterName: {}},
		ProjectId:             "projectA",
	}

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), "", &manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
			listParameterName:      listParameterValue,
			envParameterName:       envParameterValue,
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	parameters, refs, skip, errors := convertParameters(convertContext, environment, config)

	assert.Assert(t, is.Nil(errors))
	assert.Equal(t, 5, len(parameters))
	assert.Equal(t, 1, len(refs))
	assert.Equal(t, false, skip, "should not be skipped")

	nameParameter, found := parameters["name"]

	assert.Equal(t, true, found)
	assert.Equal(t, configName, nameParameter.(*valueParam.ValueParameter).Value)

	simpleParameter, found := parameters[simpleParameterName]

	assert.Equal(t, true, found)
	assert.Equal(t, simpleParameterValue, simpleParameter.(*valueParam.ValueParameter).Value)

	referenceParameter, found := parameters[referenceParameterName]

	assert.Equal(t, true, found)

	references := referenceParameter.GetReferences()

	assert.Equal(t, 1, len(references))

	ref := references[0]

	assert.Equal(t, "projectB", ref.Config.Project)
	assert.Equal(t, "management-zone", ref.Config.Api)
	assert.Equal(t, "zone", ref.Config.Config)
	assert.Equal(t, "id", ref.Property)

	listParameter, found := parameters[listParameterName]
	assert.Equal(t, true, found)
	assert.DeepEqual(t, []valueParam.ValueParameter{{"GEOLOCATION-41"}, {"GEOLOCATION-42"}, {"GEOLOCATION-43"}}, listParameter.(*listParam.ListParameter).Values)

	envParameter, found := parameters[envParameterName]
	assert.Equal(t, true, found)
	assert.DeepEqual(t, "SOME_ENV_VAR", envParameter.(*envParam.EnvironmentVariableParameter).Name)
}

func TestParseSkipDeploymentParameter(t *testing.T) {
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: "projectA",
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                                 configName,
			configv1.SkipConfigDeploymentParameter: "true",
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	testCases := []struct {
		shouldFail    bool
		testValue     string
		expectedValue bool
	}{
		{
			shouldFail:    false,
			testValue:     "true",
			expectedValue: true,
		},
		{
			shouldFail:    false,
			testValue:     "TRue",
			expectedValue: true,
		},
		{
			shouldFail:    false,
			testValue:     "false",
			expectedValue: false,
		},
		{
			shouldFail:    false,
			testValue:     "FaLse",
			expectedValue: false,
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
	}

	for _, c := range testCases {
		skip, err := parseSkipDeploymentParameter(convertContext, config, c.testValue)

		if c.shouldFail {
			assert.Assert(t, err != nil, "there should be an error for `%s`", c.testValue)
		} else {
			assert.NilError(t, err, "there should be no error for `%s`", c.testValue)
			assert.Equal(t, c.expectedValue, skip, "should be `%t` for `%s`", c.expectedValue, c.testValue)
		}
	}
}

func TestParseAbsoluteReference(t *testing.T) {
	config := generateDummyConfig(t)

	ref, err := parseReference(&ConfigConvertContext{
		ProjectId: "projectA",
	}, config, "test", "/projectB/management-zone/zone.id")

	assert.NilError(t, err)
	assert.Equal(t, "projectB", ref.Config.Project)
	assert.Equal(t, "management-zone", ref.Config.Api)
	assert.Equal(t, "zone", ref.Config.Config)
	assert.Equal(t, "id", ref.Property)
}

func TestParseRelativeReference(t *testing.T) {
	config := generateDummyConfig(t)

	ref, err := parseReference(&ConfigConvertContext{
		ProjectId: "projectA",
	}, config, "test", "projectB/management-zone/zone.id")

	assert.NilError(t, err)
	assert.Equal(t, "projectB", ref.Config.Project)
	assert.Equal(t, "management-zone", ref.Config.Api)
	assert.Equal(t, "zone", ref.Config.Config)
	assert.Equal(t, "id", ref.Property)
}

func TestParseInvalidReference(t *testing.T) {
	config := generateDummyConfig(t)

	ref, err := parseReference(&ConfigConvertContext{
		ProjectId: "projectA",
	}, config, "test", "/management-zone/zone.id")

	assert.Assert(t, is.Nil(ref))
	assert.Assert(t, err != nil)
}

func TestLoadPropertiesForEnvironment(t *testing.T) {
	environmentName := "dev"
	groupName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), groupName, &manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

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

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	envProperties := loadPropertiesForEnvironment(environment, config)

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

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		ProjectId: "projectA",
	}

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), "", &manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, config)

	assert.Equal(t, 0, len(errors), "errors: %s", errors)
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, api.GetId(), convertedConfig.Coordinate.Api)
	assert.Equal(t, configId, convertedConfig.Coordinate.Config)
	assert.Equal(t, environmentName, convertedConfig.Environment)

	references := convertedConfig.References

	assert.Equal(t, 1, len(references))
	assert.Equal(t, "projectB", references[0].Project)
	assert.Equal(t, "management-zone", references[0].Api)
	assert.Equal(t, "zone", references[0].Config)

	assert.Equal(t, 4, len(convertedConfig.Parameters))
	assert.Equal(t, configName, convertedConfig.Parameters["name"].(*valueParam.ValueParameter).Value)
	assert.Equal(t, simpleParameterValue, convertedConfig.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)
	assert.Equal(t, envVarName,
		convertedConfig.Parameters[transformEnvironmentToParamName(envVarName)].(*envParam.EnvironmentVariableParameter).Name)
}

func TestConvertDeprecatedConfigToLatest(t *testing.T) {
	projectId := "projectA"
	environmentName := "development"
	configId := "application-1"
	configName := "Application 1"
	simpleParameterValue := "hello"
	referenceParameterValue := "/projectB/management-zone/zone.id"
	envVarName := "TEST_VAR"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		ProjectId: "projectA",
	}

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), "", &manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("application", "/api/config/v1/application/web", false, "application-web", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
			simpleParameterName:    simpleParameterValue,
			referenceParameterName: referenceParameterValue,
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, config)

	assert.Equal(t, 0, len(errors), "errors: %s", errors)
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, api.IsDeprecatedBy(), convertedConfig.Coordinate.Api)
	assert.Equal(t, configId, convertedConfig.Coordinate.Config)
	assert.Equal(t, environmentName, convertedConfig.Environment)

	references := convertedConfig.References

	assert.Equal(t, 1, len(references))
	assert.Equal(t, "projectB", references[0].Project)
	assert.Equal(t, "management-zone", references[0].Api)
	assert.Equal(t, "zone", references[0].Config)

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

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		ProjectId: "projectA",
	}

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), "",
		&manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":              configName,
			simpleParameterName: simpleParameterValue,
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	_, errors := convertConfig(convertContext, environment, config)

	assert.Assert(t, len(errors) > 0, "expected errors, but got none")
}

func TestConvertSkippedConfig(t *testing.T) {
	projectId := "projectA"
	environmentName := "development"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: "projectA",
	}

	environment := manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), "", &manifest.EnvironmentVariableToken{"token"})

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                               configName,
			config.SkipConfigDeploymentParameter: "true",
		},
	}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	convertedConfig, errors := convertConfig(convertContext, environment, config)

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, projectId, convertedConfig.Coordinate.Project)
	assert.Equal(t, api.GetId(), convertedConfig.Coordinate.Api)
	assert.Equal(t, configId, convertedConfig.Coordinate.Config)
	assert.Equal(t, environmentName, convertedConfig.Environment)
	assert.Equal(t, true, convertedConfig.Skip)
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
		environmentName:  manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), environmentGroup, &manifest.EnvironmentVariableToken{"token"}),
		environmentName2: manifest.NewEnvironmentDefinition(environmentName2, createSimpleUrlDefinition(), environmentGroup2, &manifest.EnvironmentVariableToken{"token"}),
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{
		configId: {
			"name":                 configName,
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

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)
	assert.NilError(t, err)

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: projectId,
	}

	convertedConfigs, errors := convertConfigs(convertContext, environments, []configv1.Config{config})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 2, len(convertedConfigs))

	apiConfigs := convertedConfigs[environmentName]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
	assert.Equal(t, 5, len(c.Parameters))

	// assert value param is converted as expected
	assert.Equal(t, simpleParameterValue, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	// assert value param is converted as expected
	assert.Equal(t, coordinate.Coordinate{
		Project: "projectB",
		Api:     "management-zone",
		Config:  "zone",
	}, c.Parameters[referenceParameterName].(*reference.ReferenceParameter).Config)
	assert.Equal(t, "id", c.Parameters[referenceParameterName].(*reference.ReferenceParameter).Property)

	// assert list param is converted as expected
	assert.DeepEqual(t, []valueParam.ValueParameter{{"GEOLOCATION-41"}, {"GEOLOCATION-42"}, {"GEOLOCATION-43"}}, c.Parameters[listParameterName].(*listParam.ListParameter).Values)

	// assert env reference in template has created correct env parameter
	assert.Equal(t, envVariableName, c.Parameters[transformEnvironmentToParamName(envVariableName)].(*envParam.EnvironmentVariableParameter).Name)

	apiConfigs = convertedConfigs[environmentName2]
	assert.Equal(t, 1, len(apiConfigs))

	configs = apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c = configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
	assert.Equal(t, 5, len(c.Parameters))

	// assert override simple param is converted as expected
	assert.Equal(t, simpleParameterValue2, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	// assert override list param is converted as expected
	// assert list param is converted as expected
	assert.DeepEqual(t, []valueParam.ValueParameter{{"james.t.kirk@dynatrace.com"}}, c.Parameters[listParameterName].(*listParam.ListParameter).Values)
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

		environmentName:  manifest.NewEnvironmentDefinition(environmentName, createSimpleUrlDefinition(), environmentGroup, &manifest.EnvironmentVariableToken{"token"}),
		environmentName2: manifest.NewEnvironmentDefinition(environmentName2, createSimpleUrlDefinition(), environmentGroup2, &manifest.EnvironmentVariableToken{"token"}),
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

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

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	project := &projectv1.ProjectImpl{
		Id:      projectId,
		Configs: []configv1.Config{config},
	}

	projectDefinitions, convertedProjects, errors := convertProjects(convertContext, environments, []projectv1.Project{project})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 1, len(projectDefinitions))
	assert.Equal(t, 1, len(convertedProjects))

	projectDefinition := projectDefinitions[projectId]
	convertedProject := convertedProjects[0]

	assert.Equal(t, projectId, projectDefinition.Name)
	assert.Equal(t, projectId, projectDefinition.Path)
	assert.Equal(t, 2, len(convertedProject.Dependencies))

	assert.Equal(t, 1, len(convertedProject.Dependencies[environmentName]))
	assert.Equal(t, "projectB", convertedProject.Dependencies[environmentName][0])

	assert.Equal(t, 1, len(convertedProject.Dependencies[environmentName2]))
	assert.Equal(t, "projectB", convertedProject.Dependencies[environmentName2][0])

	convertedConfigs := convertedProject.Configs

	assert.Equal(t, 2, len(convertedConfigs))

	apiConfigs := convertedConfigs[environmentName]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
	assert.Equal(t, 3, len(c.Parameters))
	assert.Equal(t, simpleParameterValue, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	apiConfigs = convertedConfigs[environmentName2]
	assert.Equal(t, 1, len(apiConfigs))

	configs = apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c = configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
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

	assert.NilError(t, err)

	templ, envParams, _, errs := convertTemplate(&ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Assert(t, len(errs) == 0, "expected no errors but got %d: %s", len(errs), errs)
	assert.Assert(t, templ != nil)

	for _, env := range []string{
		"HELLO",
		"HELLO_WORLD",
		"NAME",
		"HE",
	} {
		paramName := transformEnvironmentToParamName(env)
		param, found := envParams[paramName]

		assert.Assert(t, found, "should contain `%s`", paramName)
		assert.Assert(t, param != nil, "param `%s` should be not nil", paramName)

		envParam, ok := param.(*envParam.EnvironmentVariableParameter)
		assert.Assert(t, ok, "param `%s` should be an environment variable", paramName)
		assert.Assert(t, !envParam.HasDefaultValue, "param `%s` should have no default value")
		assert.Equal(t, env, envParam.Name)
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

	assert.NilError(t, err)

	templ, _, listParamIds, errs := convertTemplate(&ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Assert(t, len(errs) == 0, "expected no errors but got %d: %s", len(errs), errs)
	assert.Assert(t, templ != nil)

	assert.Equal(t, len(listParamIds), 2, " expected to list param ids to be found in template")
	_, paramFound := listParamIds["list"]
	assert.Assert(t, paramFound)
	_, paramFound = listParamIds["list1"]
	assert.Assert(t, paramFound)
	assert.Equal(t, templ.Content(), `{
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

	assert.NilError(t, err)

	templ, envParams, listParamIds, errs := convertTemplate(&ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json", "test.json")

	assert.Assert(t, len(errs) == 0, "expected no errors but got %d: %s", len(errs), errs)
	assert.Assert(t, templ != nil)

	// check list parameter
	assert.Equal(t, len(listParamIds), 1, " expected to list param ids to be found in template")
	_, listParamFound := listParamIds["list_value"]
	assert.Assert(t, listParamFound)

	// check env parameter
	paramName := transformEnvironmentToParamName("ENV_VALUE")
	param, found := envParams[paramName]

	assert.Assert(t, found, "EnvParam should contain `%s`", paramName)
	assert.Assert(t, param != nil, "EvnParam `%s` should be not nil", paramName)

	envParam, ok := param.(*envParam.EnvironmentVariableParameter)
	assert.Assert(t, ok, "EnvParam `%s` should be an environment variable", paramName)
	assert.Assert(t, !envParam.HasDefaultValue, "EnvParam `%s` should have no default value")
	assert.Equal(t, "ENV_VALUE", envParam.Name)

	// check converted template
	assert.Equal(t, templ.Content(), `{
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
	assert.Assert(t, len(errs) == 0, "expected no errors but got %d: %s", len(errs), errs)
	assert.Equal(t, len(paramIds), 2, " expected to list param ids to be found in template")

	_, paramFound := paramIds["list"]
	assert.Assert(t, paramFound)
	_, paramFound = paramIds["list1"]
	assert.Assert(t, paramFound)

	assert.Equal(t, result, expected)
}

func setupDummyFs(t *testing.T) afero.Fs {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, "test/test-config.json", []byte(`{}`), 0644)

	assert.NilError(t, err)

	return fs
}

func setupDummyFsWithEnvVariableInTemplate(t *testing.T, envVarName string) afero.Fs {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, "test/test-config.json", []byte(fmt.Sprintf(`{"test": "{{.Env.%s}}"}`, envVarName)), 0644)

	assert.NilError(t, err)

	return fs
}

func setupFsWithFullTestTemplate(t *testing.T, simpleVar, refVar, listVar, envVar string) (afero.Fs, util.Template) {
	fs := afero.NewMemMapFs()

	err := fs.Mkdir("test", 0644)
	assert.NilError(t, err)

	templateContent := fmt.Sprintf(`{ "simple": "{{ .%s }}", "reference": "{{ .%s }}", "list": [ {{ .%s }} ], "env": "{{ .Env.%s }}" }`, simpleVar, refVar, listVar, envVar)

	template, err := util.NewTemplateFromString("test/test-config.json", templateContent)
	assert.NilError(t, err)

	err = afero.WriteFile(fs, "test/test-config.json", []byte(templateContent), 0644)
	assert.NilError(t, err)

	return fs, template
}

func generateDummyTemplate(t *testing.T) util.Template {
	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	return template
}

func generateDummyConfig(t *testing.T) configv1.Config {
	var configId = "alerting-profile-1"

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false, "", false)

	properties := map[string]map[string]string{}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	conf, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	return conf
}

func TestAdjustProjectId(t *testing.T) {
	id := adjustProjectId(`test\project/name`)

	assert.Equal(t, `test.project.name`, id)
}

func createSimpleUrlDefinition() manifest.UrlDefinition {
	return manifest.UrlDefinition{
		Type:  manifest.ValueUrlType,
		Value: "test.env",
	}
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
