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

// +build unit

package converter

import (
	"fmt"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	configv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	envParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/environment"
	valueParam "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestConvertParameters(t *testing.T) {
	environmentName := "test"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterName := "randomValue"
	simpleParameterValue := "hello"
	referenceParameterName := "managementZoneId"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: "projectA",
	}

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		Url:   "test.env",
		Group: "",
		Token: &manifest.EnvironmentVariableToken{
			EnvironmentVariableName: "token",
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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

	parameters, refs, skip, errors := convertParameters(convertContext, environment, config)

	assert.Assert(t, is.Nil(errors))
	assert.Equal(t, 3, len(parameters))
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

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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
	simpleParameterName := "randomValue"
	simpleParameterValue := "hello"
	referenceParameterName := "managementZoneId"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		Url:   "test.env",
		Group: groupName,
		Token: &manifest.EnvironmentVariableToken{
			EnvironmentVariableName: "token",
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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
	simpleParameterName := "randomValue"
	simpleParameterValue := "hello"
	referenceParameterName := "managementZoneId"
	referenceParameterValue := "/projectB/management-zone/zone.id"
	envVarName := "TEST_VAR"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFsWithEnvVariableInTemplate(t, envVarName),
		},
		ProjectId: "projectA",
	}

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		Url:   "test.env",
		Group: "",
		Token: &manifest.EnvironmentVariableToken{
			EnvironmentVariableName: "token",
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		Url:   "test.env",
		Group: "",
		Token: &manifest.EnvironmentVariableToken{
			EnvironmentVariableName: "token",
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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

	environment := manifest.EnvironmentDefinition{
		Name:  environmentName,
		Url:   "test.env",
		Group: "",
		Token: &manifest.EnvironmentVariableToken{
			EnvironmentVariableName: "token",
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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
	simpleParameterName := "randomValue"
	simpleParameterValue := "hello"
	simpleParameterValue2 := "world"

	convertContext := &ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: setupDummyFs(t),
		},
		ProjectId: projectId,
	}

	environments := map[string]manifest.EnvironmentDefinition{
		environmentName: {
			Name:  environmentName,
			Url:   "test.env",
			Group: environmentGroup,
			Token: &manifest.EnvironmentVariableToken{
				EnvironmentVariableName: "token",
			},
		},

		environmentName2: {
			Name:  environmentName2,
			Url:   "test.env",
			Group: environmentGroup2,
			Token: &manifest.EnvironmentVariableToken{
				EnvironmentVariableName: "token",
			},
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

	properties := map[string]map[string]string{
		configId: {
			"name":              configName,
			simpleParameterName: simpleParameterValue,
		},
		configId + "." + environmentGroup2: {
			simpleParameterName: simpleParameterValue2,
		},
	}

	template := generateDummyTemplate(t)

	config, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	convertedConfigs, errors := convertConfigs(convertContext, environments, []configv1.Config{config})

	assert.Equal(t, 0, len(errors))
	assert.Equal(t, 2, len(convertedConfigs))

	apiConfigs := convertedConfigs[environmentName]
	assert.Equal(t, 1, len(apiConfigs))

	configs := apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c := configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
	assert.Equal(t, 2, len(c.Parameters))
	assert.Equal(t, simpleParameterValue, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)

	apiConfigs = convertedConfigs[environmentName2]
	assert.Equal(t, 1, len(apiConfigs))

	configs = apiConfigs[api.GetId()]
	assert.Equal(t, 1, len(configs))

	c = configs[0]
	assert.Equal(t, configId, c.Coordinate.Config)
	assert.Equal(t, 2, len(c.Parameters))
	assert.Equal(t, simpleParameterValue2, c.Parameters[simpleParameterName].(*valueParam.ValueParameter).Value)
}

func TestConvertProjects(t *testing.T) {
	projectId := "projectA"
	environmentName := "dev"
	environmentGroup := "development"
	environmentName2 := "sprint"
	environmentGroup2 := "hardening"
	configId := "alerting-profile-1"
	configName := "Alerting Profile 1"
	simpleParameterName := "randomValue"
	simpleParameterValue := "hello"
	simpleParameterValue2 := "world"
	referenceParameterName := "managementZoneId"
	referenceParameterValue := "/projectB/management-zone/zone.id"

	convertContext := &ConverterContext{
		Fs: setupDummyFs(t),
	}

	environments := map[string]manifest.EnvironmentDefinition{
		environmentName: {
			Name:  environmentName,
			Url:   "test.env",
			Group: environmentGroup,
			Token: &manifest.EnvironmentVariableToken{
				EnvironmentVariableName: "token",
			},
		},

		environmentName2: {
			Name:  environmentName2,
			Url:   "test.env",
			Group: environmentGroup2,
			Token: &manifest.EnvironmentVariableToken{
				EnvironmentVariableName: "token",
			},
		},
	}

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

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

func TestConvertTemplate(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, "test.json", []byte(`{
		"test": "{{.Env.HELLO}}",
		"test1": "{{ .Env.HELLO }}",
		"test2": "{{  .Env.HELLO_WORLD}} {{ .Env.NAME }}",
		"test3": "{{  .Env.HELLO_WORLD}} {{ .Env.HE     }}",
	}`), 0644)

	assert.NilError(t, err)

	templ, envParams, errs := convertTemplate(&ConfigConvertContext{
		ConverterContext: &ConverterContext{
			Fs: fs,
		},
		ProjectId: "projectA",
	}, "test.json")

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

func generateDummyTemplate(t *testing.T) util.Template {
	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	return template
}

func generateDummyConfig(t *testing.T) configv1.Config {
	var configId = "alerting-profile-1"

	api := api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles")

	properties := map[string]map[string]string{}

	template, err := util.NewTemplateFromString("test/test-config.json", "{}")

	assert.NilError(t, err)

	conf, err := configv1.NewConfigWithTemplate(configId, "test-project", "test/test-config.json",
		template, properties, api)

	assert.NilError(t, err)

	return conf
}
