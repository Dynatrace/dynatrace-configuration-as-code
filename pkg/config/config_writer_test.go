//go:build unit

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

package config

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/internal/persistence"
	envParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/stretchr/testify/assert"
)

func TestExtractCommonBase(t *testing.T) {
	configName := "test-config-1"
	group := "development"
	template := "test.json"

	param1Name := "config number"
	param1Value := "12"

	param2Name := "dashboardId"
	param2Value := []interface{}{"projectA", "dashboard", "important", "id"}

	param3Name := "dashboardId2"
	param3Value := map[interface{}]interface{}{
		"type":     "reference",
		"project":  "projectA",
		"api":      "dashboard",
		"config":   "test",
		"property": "id",
	}

	parameterNotSharedName := "not-shared"

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: "not-shared",
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 12,
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test1",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 25,
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test2",
		},
	}

	base, rest := extractCommonBase(configs)

	assert.NotNil(t, base, "there should be a common base")

	assert.Equal(t, base.Name, configName, "name should be `%s`, but was `%s`", configName, base.Name)
	assert.Equal(t, base.Template, template, "template should be `%s`, but was `%s`", template, base.Template)
	assert.Nil(t, base.Skip, "skip should be nil: %v", base.Skip)
	assert.Len(t, base.Parameters, 3, "there should be 3 parameter overrides, but there were `%d`",
		len(base.Parameters))

	for _, n := range []string{param1Name, param2Name, param3Name} {
		param := base.Parameters[n]
		assert.NotNil(t, param, "`%s` should be present in base", n)
	}

	assert.Len(t, rest, 3, "there should be `3` overrides, but there were `%d`", len(rest))

	for _, r := range rest {
		for _, n := range []string{param1Name, param2Name, param3Name} {
			param := r.Parameters[n]
			assert.Nil(t, param, "`%s` should not be present in override for `%s`", n, r.environment)
		}
	}
}

func TestExtractCommonBaseForEnvVarSkipsWithEqualValues(t *testing.T) {
	configName := "test-config-1"
	group := "development"
	template := "test.json"

	param1Name := "config number"
	param1Value := "12"

	param2Name := "dashboardId"
	param2Value := []interface{}{"projectA", "dashboard", "important", "id"}

	param3Name := "dashboardId2"
	param3Value := map[interface{}]interface{}{
		"type":     "reference",
		"project":  "projectA",
		"api":      "dashboard",
		"config":   "test",
		"property": "id",
	}

	parameterNotSharedName := "not-shared"

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 12,
				},
				Template: template,
				Skip: map[any]any{
					"type": "environment",
					"name": "A",
				},
			},
			group:       group,
			environment: "test",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 13,
				},
				Template: template,
				Skip: map[any]any{
					"type": "environment",
					"name": "A",
				},
			},
			group:       group,
			environment: "test1",
		},
	}

	base, rest := extractCommonBase(configs)

	assert.NotNil(t, base, nil, "there should be a common base")

	assert.Equal(t, base.Name, configName, "name should be `%s`, but was `%s`", configName, base.Name)
	assert.Equal(t, base.Template, template, "template should be `%s`, but was `%s`", template, base.Template)
	assert.NotNil(t, base.Skip, "skip should not be nil")
	assert.Len(t, base.Parameters, 3, "there should be 3 base-parameters, but there were `%d`", len(base.Parameters))

	for _, n := range []string{param1Name, param2Name, param3Name} {
		param := base.Parameters[n]
		assert.NotNil(t, param, "`%s` should be present in base", n)
	}

	assert.NotNil(t, base.Skip, "skip should be in the base")

	assert.Equal(t, base.Skip, map[any]any{
		"type": "environment",
		"name": "A",
	})

	assert.Len(t, rest, 2, "there should be `2` overrides, but there were `%d`", len(rest))

	for _, r := range rest {
		for _, n := range []string{param1Name, param2Name, param3Name} {
			param := r.Parameters[n]
			assert.Nil(t, param, "`%s` should not be present in override for `%s`", n, r.environment)
		}
	}
}

func TestExtractCommonBaseForEnvVarSkipsWithDifferentValues(t *testing.T) {
	configName := "test-config-1"
	group := "development"
	template := "test.json"

	param1Name := "config number"
	param1Value := "12"

	param2Name := "dashboardId"
	param2Value := []interface{}{"projectA", "dashboard", "important", "id"}

	param3Name := "dashboardId2"
	param3Value := map[interface{}]interface{}{
		"type":     "reference",
		"project":  "projectA",
		"api":      "dashboard",
		"config":   "test",
		"property": "id",
	}

	skipA := map[any]any{
		"type": "environment",
		"name": "A",
	}
	skipB := map[any]any{
		"type": "environment",
		"name": "B",
	}
	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name: param1Value,
					param2Name: param2Value,
					param3Name: param3Value,
				},
				Template: template,
				Skip:     skipA,
			},
			group:       group,
			environment: "test",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name: param1Value,
					param2Name: param2Value,
					param3Name: param3Value,
				},
				Template: template,
				Skip:     skipB,
			},
			group:       group,
			environment: "test1",
		},
	}

	base, rest := extractCommonBase(configs)

	assert.NotNil(t, base, "there should be a common base")

	assert.Equal(t, base.Name, configName, "name should be `%s`, but was `%s`", configName, base.Name)
	assert.Equal(t, base.Template, template, "template should be `%s`, but was `%s`", template, base.Template)
	assert.Nil(t, base.Skip, "base skip should be nil")
	assert.Len(t, base.Parameters, 3, "there should be 3 base-parameters, but there were `%d`", len(base.Parameters))

	for _, n := range []string{param1Name, param2Name, param3Name} {
		param := base.Parameters[n]
		assert.NotNil(t, param, "`%s` should be present in base", n)
	}

	assert.Len(t, rest, 2, "there should be `2` overrides, but there were `%d`", len(rest))

	for _, r := range rest {
		for _, n := range []string{param1Name, param2Name, param3Name} {
			param := r.Parameters[n]
			assert.Nil(t, param, "`%s` should not be present in override for `%s`", n, r.environment)
		}
	}

	assert.Equal(t, rest[0].Skip, skipA)
	assert.Equal(t, rest[1].Skip, skipB)
}

func TestExtractCommonBaseT(t *testing.T) {
	configName := "test-config-1"
	group := "development"
	template := "test.json"

	param1Name := "config number"
	param1Value := "12"

	param2Name := "dashboardId"
	param2Value := []interface{}{"projectA", "dashboard", "important", "id"}

	param3Name := "dashboardId2"
	param3Value := map[interface{}]interface{}{
		"type":     "reference",
		"project":  "projectA",
		"api":      "dashboard",
		"config":   "test",
		"property": "id",
	}

	parameterNotSharedName := "not-shared"

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: "not-shared",
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 12,
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test1",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name:             param1Value,
					param2Name:             param2Value,
					param3Name:             param3Value,
					parameterNotSharedName: 25,
				},
				Template: template,
				Skip:     true,
			},
			group:       group,
			environment: "test2",
		},
	}

	base, rest := extractCommonBase(configs)

	assert.NotNil(t, base, "there should be a common base")

	assert.Equal(t, base.Name, configName, "name should be `%s`, but was `%s`", configName, base.Name)
	assert.Equal(t, base.Template, template, "template should be `%s`, but was `%s`", template, base.Template)
	assert.Nil(t, base.Skip, "skip should be nil: %v", base.Skip)
	assert.Len(t, base.Parameters, 3, "there should be 3 parameter overrides, but there were `%d`",
		len(base.Parameters))

	for _, n := range []string{param1Name, param2Name, param3Name} {
		param := base.Parameters[n]
		assert.NotNil(t, param, "`%s` should be present in base", n)
	}

	assert.Len(t, rest, 3, "there should be `3` overrides, but there were `%d`", len(rest))

	for _, r := range rest {
		for _, n := range []string{param1Name, param2Name, param3Name} {
			param := r.Parameters[n]
			assert.Nil(t, param, "`%s` should not be present in override for `%s`", n, r.environment)
		}
	}
}

func TestExtractCommonBaseWithJustSkipDifferent(t *testing.T) {
	configName := "test-config-1"
	group := "development"
	template := "test.json"

	param1Name := "config number"
	param1Value := "12"

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name: param1Value,
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name: param1Value,
				},
				Template: template,
				Skip:     true,
			},
			group:       group,
			environment: "test1",
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: configName,
				Parameters: map[string]persistence.ConfigParameter{
					param1Name: param1Value,
				},
				Template: template,
				Skip:     nil,
			},
			group:       group,
			environment: "test2",
		},
	}

	base, rest := extractCommonBase(configs)

	assert.NotNil(t, base, "there should be a common base")

	assert.Equal(t, base.Name, configName, "name should be `%s`, but was `%s`", configName, base.Name)
	assert.Equal(t, base.Template, template, "template should be `%s`, but was `%s`", template, base.Template)
	assert.Nil(t, base.Skip, "skip should be nil: %v", base.Skip)
	assert.Len(t, base.Parameters, 1, "there should be 1 parameter overrides, but there were `%d`",
		len(base.Parameters))

	assert.NotNil(t, base.Parameters[param1Name], "`%s` should be present in base", param1Name)

	assert.Len(t, rest, 3, "there should be `3` overrides, but there were `%d`", len(rest))

	for _, r := range rest {
		assert.Nil(t, r.Parameters[param1Name], "`%s` should not be present in override for `%s`",
			param1Name, r.environment)
	}
}

func TestToParameterDefinition(t *testing.T) {
	paramName := "test-param-1"
	paramValue := "hello"

	context := detailedSerializerContext{
		serializerContext: &serializerContext{
			WriterContext: &WriterContext{
				ParametersSerde: map[string]parameter.ParameterSerDe{
					parameter.DummyParameterType: {
						Serializer: func(c parameter.ParameterWriterContext) (map[string]interface{}, error) {
							return map[string]interface{}{
								"Value": c.Parameter.(*parameter.DummyParameter).Value,
							}, nil
						},
					},
				},
			},
		},
	}

	result, err := toParameterDefinition(&context, paramName, &parameter.DummyParameter{
		Value: paramValue,
	})

	assert.NoError(t, err, "to parameter definiton should return no error, but was `%s`", err)
	assert.NotNil(t, result, "result should not be nil")

	resultMap, ok := result.(map[string]interface{})

	assert.True(t, ok, "result should be a map")
	assert.Equal(t, resultMap["Value"], "hello", "result should have key `Value` with value `%s`, but was `%s`",
		paramValue, resultMap["Value"])
}

func TestToParameterDefinitionShouldDoSpecialParameterDefinitionIfActivatedAndSupported(t *testing.T) {
	paramName := "test-param-1"
	paramValue := "hello"

	context := detailedSerializerContext{
		serializerContext: &serializerContext{
			WriterContext: &WriterContext{},
		},
	}

	result, err := toParameterDefinition(&context, paramName, &value.ValueParameter{
		Value: paramValue,
	})

	assert.NoError(t, err, "to parameter definiton should return no error: %s", err)
	assert.NotNil(t, result, "result should not be nil")

	assert.Equal(t, result, paramValue, "result should be value `%s`, but was `%v`", paramValue, result)
}

func TestToParameterDefinitionShouldWithShortSyntaxActiveShouldDoNormalWhenParameterIsMap(t *testing.T) {
	paramName := "test-param-1"
	paramValue := map[string]interface{}{
		"name": "hansi",
	}

	context := detailedSerializerContext{
		serializerContext: &serializerContext{
			WriterContext: &WriterContext{
				ParametersSerde: map[string]parameter.ParameterSerDe{
					value.ValueParameterType: value.ValueParameterSerde,
				},
			},
		},
	}

	result, err := toParameterDefinition(&context, paramName, &value.ValueParameter{
		Value: paramValue,
	})

	assert.NoError(t, err, "to parameter definiton should return no error: %s", err)
	assert.NotNil(t, result, "result should not be nil")

	resultMap, ok := result.(map[string]interface{})

	assert.True(t, ok, "result should be map")
	assert.Equal(t, resultMap["type"], value.ValueParameterType, "result map should be of type `%s`, but was `%s`",
		value.ValueParameterType, resultMap["type"])
	assert.NotNil(t, resultMap["value"], "result map should contain a 'value' entry")
	assert.Equal(t, resultMap["value"], paramValue)
}

func TestForSamePropertiesWithNothingSet(t *testing.T) {
	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     nil,
				Template: "",
				Skip:     nil,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     nil,
				Template: "",
				Skip:     nil,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     nil,
				Template: "",
				Skip:     nil,
			},
		},
	}

	result := testForSameProperties(configs)

	assertPropertyCheckResult(t, propertyCheckResult{
		shareName: true,
		foundName: false,
		name:      nil,

		shareTemplate: true,
		foundTemplate: false,
		template:      "",

		shareSkip: true,
		foundSkip: false,
		skip:      nil,
	}, result)
}

func TestForSamePropertiesWithAllShared(t *testing.T) {
	name := "name"
	template := "test.json"
	skip := false

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     name,
				Template: template,
				Skip:     skip,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     name,
				Template: template,
				Skip:     skip,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name:     name,
				Template: template,
				Skip:     skip,
			},
		},
	}

	result := testForSameProperties(configs)

	assertPropertyCheckResult(t, propertyCheckResult{
		shareName: true,
		foundName: true,
		name:      name,

		shareTemplate: true,
		foundTemplate: true,
		template:      template,

		shareSkip: true,
		foundSkip: true,
		skip:      skip,
	}, result)
}

func TestForSamePropertiesWithNameNotSharedByAll(t *testing.T) {
	sharedName := "name"

	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: sharedName,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: nil,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Name: sharedName,
			},
		},
	}

	result := testForSameProperties(configs)

	assertPropertyCheckResult(t, propertyCheckResult{
		shareName: false,
		foundName: true,

		shareTemplate: true,
		shareSkip:     true,
	}, result)
}

func TestForSamePropertiesWithSkipNotSetExceptForOne(t *testing.T) {
	configs := []extendedConfigDefinition{
		{
			ConfigDefinition: persistence.ConfigDefinition{},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{
				Skip: true,
			},
		},
		{
			ConfigDefinition: persistence.ConfigDefinition{},
		},
	}

	result := testForSameProperties(configs)

	assertPropertyCheckResult(t, propertyCheckResult{
		shareName:     true,
		shareTemplate: true,

		shareSkip: false,
		foundSkip: true,
	}, result)
}

func assertPropertyCheckResult(t *testing.T, expected propertyCheckResult, actual propertyCheckResult) {
	assert.Equal(t, expected.foundName, actual.foundName)
	assert.Equal(t, expected.foundTemplate, actual.foundTemplate)
	assert.Equal(t, expected.foundSkip, actual.foundSkip)

	assert.Equal(t, expected.shareName, actual.shareName)
	assert.Equal(t, expected.shareTemplate, actual.shareTemplate)
	assert.Equal(t, expected.shareSkip, actual.shareSkip)

	assert.Equal(t, expected.name, actual.name)
	assert.Equal(t, expected.template, actual.template)
	assert.Equal(t, expected.skip, actual.skip)
}

func TestWriteConfigs(t *testing.T) {

	var tests = []struct {
		name                  string
		configs               []Config
		expectedConfigs       map[string]persistence.TopLevelDefinition
		expectedTemplatePaths []string
		expectedErrs          []string
	}{
		{
			name: "Simple classic API write",
			configs: []Config{
				{
					Template: template.CreateTemplateFromString("project/alerting-profile/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "alerting-profile",
						ConfigId: "configId",
					},
					Type: ClassicApiType{
						Api: "alerting-profile",
					},
					Parameters: map[string]parameter.Parameter{
						NameParameter: &value.ValueParameter{Value: "name"},
					},
					SkipForConversion: envParam.New("ENV_VAR_SKIP"),
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"alerting-profile": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip: map[any]any{
									"type": "environment",
									"name": "ENV_VAR_SKIP",
								},
							},
							Type: persistence.TypeDefinition{
								Api: "alerting-profile",
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/alerting-profile/a.json",
				"project/alerting-profile/config.yaml",
			},
		},
		{
			name: "Settings 2.0 schema write sanitizes names",
			configs: []Config{
				{
					Template: template.NewDownloadTemplate("a", "", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:alerting-profile",
						ConfigId: "configId",
					},
					Type: SettingsType{
						SchemaId: "builtin:alerting-profile",
					},
					Parameters: map[string]parameter.Parameter{
						NameParameter:  &value.ValueParameter{Value: "name"},
						ScopeParameter: value.New("tenant"),
					},
					SkipForConversion: value.New("true"),
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"builtinalerting-profile": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       "true",
							},
							Type: persistence.TypeDefinition{
								Settings: persistence.SettingsDefinition{
									Schema: "builtin:alerting-profile",
									Scope:  "tenant",
								},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/builtinalerting-profile/config.yaml",
				"project/builtinalerting-profile/a.json",
			},
		},
		{
			name: "Simple settings 2.0 write",
			configs: []Config{
				{
					Template: template.CreateTemplateFromString("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						ScopeParameter: &value.ValueParameter{Value: "scope"},
						NameParameter:  &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"schemaid": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       true,
							},
							Type: persistence.TypeDefinition{
								Settings: persistence.SettingsDefinition{
									Schema:        "schemaid",
									SchemaVersion: "1.2.3",
									Scope:         "scope",
								},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/schemaid/a.json",
			},
		},
		{
			name: "Automation resources",
			configs: []Config{
				{
					Template: template.CreateTemplateFromString("project/workflow/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "workflow",
						ConfigId: "configId1",
					},
					Type: AutomationType{
						Resource: Workflow,
					},
					Parameters: map[string]parameter.Parameter{
						NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.CreateTemplateFromString("project/business-calendar/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "business-calendar",
						ConfigId: "configId2",
					},
					Type: AutomationType{
						Resource: BusinessCalendar,
					},
					Parameters: map[string]parameter.Parameter{
						NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.CreateTemplateFromString("project/scheduling-rule/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "scheduling-rule",
						ConfigId: "configId3",
					},
					Type: AutomationType{
						Resource: SchedulingRule,
					},
					Parameters: map[string]parameter.Parameter{
						NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"workflow": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId1",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       true,
							},
							Type: persistence.TypeDefinition{
								Automation: persistence.AutomationDefinition{
									Resource: "workflow",
								},
							},
						},
					},
				},
				"business-calendar": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId2",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       true,
							},
							Type: persistence.TypeDefinition{
								Automation: persistence.AutomationDefinition{
									Resource: "business-calendar",
								},
							},
						},
					},
				},
				"scheduling-rule": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId3",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       true,
							},
							Type: persistence.TypeDefinition{
								Automation: persistence.AutomationDefinition{
									Resource: "scheduling-rule",
								},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/workflow/a.json",
				"project/business-calendar/a.json",
				"project/scheduling-rule/a.json",
			},
		},
		{
			name: "Reference scope",
			configs: []Config{
				{
					Template: template.CreateTemplateFromString("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						ScopeParameter: refParam.New("otherproject", "type", "id", "prop"),
						NameParameter:  &value.ValueParameter{Value: "name"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"schemaid": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "a.json",
								Skip:       false,
							},
							Type: persistence.TypeDefinition{
								Settings: persistence.SettingsDefinition{
									Schema:        "schemaid",
									SchemaVersion: "1.2.3",
									Scope: map[any]any{
										"type":       "reference",
										"configType": "type",
										"project":    "otherproject",
										"property":   "prop",
										"configId":   "id",
									},
								},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/schemaid/a.json",
			},
		},
		{
			name: "OS path separators are replaced with slashes",
			configs: []Config{
				{
					Template: template.CreateTemplateFromString(filepath.Join("general", "schemaid", "a.json"), ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						ScopeParameter: value.New("scope"),
						NameParameter:  &value.ValueParameter{Value: "name"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"schemaid": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId",
							Config: persistence.ConfigDefinition{
								Name:       "name",
								Parameters: nil,
								Template:   "../../general/schemaid/a.json",
								Skip:       false,
							},
							Type: persistence.TypeDefinition{
								Settings: persistence.SettingsDefinition{
									Schema:        "schemaid",
									SchemaVersion: "1.2.3",
									Scope:         "scope",
								},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"general/schemaid/a.json",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			fs := testutils.TempFs(t)

			errs := WriteConfigs(&WriterContext{
				Fs:              fs,
				OutputFolder:    "test",
				ProjectFolder:   "project",
				ParametersSerde: DefaultParameterParsers,
			}, tc.configs)
			errutils.PrintErrors(errs)
			assert.Equal(t, len(errs), len(tc.expectedErrs), "Produced errors do not match expected errors")

			for i := range tc.expectedErrs {
				assert.ErrorContains(t, errs[i], tc.expectedErrs[i])
			}

			// check all api-folders config file
			for apiType, definition := range tc.expectedConfigs {

				content, err := afero.ReadFile(fs, "test/project/"+apiType+"/config.yaml")
				assert.NoError(t, err, "reading config file should not produce an error")

				var s persistence.TopLevelDefinition
				err = yaml.Unmarshal(content, &s)
				assert.NoError(t, err, "unmarshalling config file should not produce an error")

				assert.Equal(t, s, definition)
			}

			// check that templates have been created
			for _, path := range tc.expectedTemplatePaths {
				expectedPath := filepath.Join("test", path)
				found, err := afero.Exists(fs, expectedPath)
				assert.NoError(t, err)
				assert.Equal(t, found, true, "could not find %q", expectedPath)
			}

		})
	}
}

func TestOrderedConfigs(t *testing.T) {
	configs := []Config{
		{
			Template:   template.CreateTemplateFromString("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "b"},
			Type:       ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{NameParameter: &value.ValueParameter{Value: "name"}},
		},
		{
			Template:   template.CreateTemplateFromString("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "a"},
			Type:       ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{NameParameter: &value.ValueParameter{Value: "name"}},
		},
		{
			Template:   template.CreateTemplateFromString("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "c"},
			Type:       ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{NameParameter: &value.ValueParameter{Value: "name"}},
		},
	}

	fs := testutils.TempFs(t)

	errs := WriteConfigs(&WriterContext{
		Fs:              fs,
		OutputFolder:    "test",
		ProjectFolder:   "project",
		ParametersSerde: DefaultParameterParsers,
	}, configs)
	assert.NoError(t, errors.Join(errs...))

	content, err := afero.ReadFile(fs, "test/project/alerting-profile/config.yaml")
	assert.NoError(t, err, "reading config file should not produce an error")

	var s persistence.TopLevelDefinition
	err = yaml.Unmarshal(content, &s)
	assert.NoError(t, err, "unmarshalling config file should not produce an error")

	// check if configs are ordered by id
	for i := 0; i < len(s.Configs)-1; i++ {
		a := s.Configs[i].Id
		b := s.Configs[i+1].Id
		assert.Less(t, a, b, "not in order: %q should be < than %q", a, b)
	}

}
