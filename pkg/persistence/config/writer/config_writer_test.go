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

package writer

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/pointer"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/internal/persistence"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
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
		configs               []config.Config
		expectedConfigs       map[string]persistence.TopLevelDefinition
		expectedTemplatePaths []string
		expectedErrs          []string
		envVars               map[string]string
	}{
		{
			name: "Simple classic API write",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/alerting-profile/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "alerting-profile",
						ConfigId: "configId",
					},
					Type: config.ClassicApiType{
						Api: "alerting-profile",
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
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
								Skip:       false,
							},
							Type: persistence.TypeDefinition{
								Type: config.ClassicApiType{
									Api: "alerting-profile",
								},
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
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplate("somethingTooLongaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "builtin:alerting-profile",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId: "builtin:alerting-profile",
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter:  &value.ValueParameter{Value: "name"},
						config.ScopeParameter: value.New("tenant"),
					},
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
								Template:   "somethingTooLongaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.json",
								Skip:       false,
							},
							Type: persistence.TypeDefinition{
								Type: config.SettingsType{
									SchemaId: "builtin:alerting-profile",
								},
								Scope: "tenant",
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/builtinalerting-profile/config.yaml",
				"project/builtinalerting-profile/somethingTooLongaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa.json",
			},
		},
		{
			name: "Simple settings 2.0 write",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "scope"},
						config.NameParameter:  &value.ValueParameter{Value: "name"},
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
								Type: config.SettingsType{
									SchemaId:      "schemaid",
									SchemaVersion: "1.2.3",
								},
								Scope: "scope",
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
			name:    "Simple settings 2.0 write with all-user permissions FF on",
			envVars: map[string]string{featureflags.AccessControlSettings.EnvName(): "true"},
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId:          "schemaid",
						SchemaVersion:     "1.2.3",
						AllUserPermission: pointer.Pointer(config.ReadPermission),
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "scope"},
						config.NameParameter:  &value.ValueParameter{Value: "name"},
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
								Type: config.SettingsType{
									SchemaId:          "schemaid",
									SchemaVersion:     "1.2.3",
									AllUserPermission: pointer.Pointer(config.ReadPermission),
								},
								Scope: "scope",
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
			name: "Simple settings 2.0 write with all-user permissions with FF off",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId:          "schemaid",
						SchemaVersion:     "1.2.3",
						AllUserPermission: pointer.Pointer(config.ReadPermission),
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: &value.ValueParameter{Value: "scope"},
						config.NameParameter:  &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
			},
			envVars: map[string]string{featureflags.AccessControlSettings.EnvName(): "false"},
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
								Type: config.SettingsType{
									SchemaId:      "schemaid",
									SchemaVersion: "1.2.3",
								},
								Scope: "scope",
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
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/workflow/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "workflow",
						ConfigId: "configId1",
					},
					Type: config.AutomationType{
						Resource: config.Workflow,
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.NewInMemoryTemplateWithPath("project/business-calendar/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "business-calendar",
						ConfigId: "configId2",
					},
					Type: config.AutomationType{
						Resource: config.BusinessCalendar,
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.NewInMemoryTemplateWithPath("project/scheduling-rule/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "scheduling-rule",
						ConfigId: "configId3",
					},
					Type: config.AutomationType{
						Resource: config.SchedulingRule,
					},
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
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
								Type: config.AutomationType{
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
								Type: config.AutomationType{
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
								Type: config.AutomationType{
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
			name: "Grail Buckets",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/bucket/mybucket.json", "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "bucket",
						ConfigId: "configId1",
					},
					Type: config.BucketType{},
					Parameters: map[string]parameter.Parameter{
						"some param": &value.ValueParameter{Value: "some value"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"bucket": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId1",
							Config: persistence.ConfigDefinition{
								Parameters: map[string]persistence.ConfigParameter{
									"some param": "some value",
								},
								Template: "mybucket.json",
								Skip:     false,
							},
							Type: persistence.TypeDefinition{
								Type: config.BucketType{},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/bucket/mybucket.json",
			},
		},
		{
			name:    "Segment",
			envVars: map[string]string{featureflags.Segments.EnvName(): "true"},
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/segment/template.json", "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "segment",
						ConfigId: "configId1",
					},
					Type: config.Segment{},
					Parameters: map[string]parameter.Parameter{
						"some param": &value.ValueParameter{Value: "some value"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"segment": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId1",
							Config: persistence.ConfigDefinition{
								Parameters: map[string]persistence.ConfigParameter{
									"some param": "some value",
								},
								Template: "template.json",
								Skip:     false,
							},
							Type: persistence.TypeDefinition{
								Type: config.Segment{},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/segment/template.json",
			},
		},
		{
			name:    "Segment should fail if FF MONACO_FEAT_SEGMENTS is not set",
			envVars: map[string]string{featureflags.Segments.EnvName(): "false"},
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/segment/template.json", "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "segment",
						ConfigId: "configId1",
					},
					Type: config.Segment{},
					Parameters: map[string]parameter.Parameter{
						"some param": &value.ValueParameter{Value: "some value"},
					},
					Skip: false,
				},
			},
			expectedErrs: []string{"config.Segment"},
		},
		{
			name:    "SLO resource",
			envVars: map[string]string{featureflags.ServiceLevelObjective.EnvName(): "true"},
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/slo_v2/template.json", "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "slo-v2",
						ConfigId: "configId1",
					},
					Type: config.ServiceLevelObjective{},
					Parameters: map[string]parameter.Parameter{
						"some param": &value.ValueParameter{Value: "some value"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"slo-v2": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId1",
							Config: persistence.ConfigDefinition{
								Parameters: map[string]persistence.ConfigParameter{
									"some param": "some value",
								},
								Template: "../slo_v2/template.json",
								Skip:     false,
							},
							Type: persistence.TypeDefinition{
								Type: config.ServiceLevelObjective{},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/slo_v2/template.json",
			},
		},
		{
			name:    "SLO with FF off, should return error",
			envVars: map[string]string{featureflags.ServiceLevelObjective.EnvName(): "false"},
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/slo_v2/template.json", "{}"),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "slo-v2",
						ConfigId: "configId1",
					},
					Type: config.ServiceLevelObjective{},
					Parameters: map[string]parameter.Parameter{
						"some param": &value.ValueParameter{Value: "some value"},
					},
					Skip: false,
				},
			},
			expectedErrs: []string{"config.ServiceLevelObjective"},
		},

		{
			name: "Reference scope",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/schemaid/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: refParam.New("otherproject", "type", "id", "prop"),
						config.NameParameter:  &value.ValueParameter{Value: "name"},
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
								Type: config.SettingsType{
									SchemaId:      "schemaid",
									SchemaVersion: "1.2.3",
								},
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
			expectedTemplatePaths: []string{
				"project/schemaid/a.json",
			},
		},
		{
			name: "OS path separators are replaced with slashes",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath(filepath.Join("general", "schemaid", "a.json"), ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.SettingsType{
						SchemaId:      "schemaid",
						SchemaVersion: "1.2.3",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: value.New("scope"),
						config.NameParameter:  &value.ValueParameter{Value: "name"},
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
								Type: config.SettingsType{
									SchemaId:      "schemaid",
									SchemaVersion: "1.2.3",
								},
								Scope: "scope",
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"general/schemaid/a.json",
			},
		},
		{
			name: "API with sub-path is persisted correctly",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath(filepath.Join("general", "alerting-profile", "a.json"), ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "schemaid",
						ConfigId: "configId",
					},
					Type: config.ClassicApiType{
						Api: "alerting-profile",
					},
					Parameters: map[string]parameter.Parameter{
						config.ScopeParameter: value.New("scope"),
						config.NameParameter:  &value.ValueParameter{Value: "name"},
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
								Template:   "../../general/alerting-profile/a.json",
								Skip:       false,
							},
							Type: persistence.TypeDefinition{
								Type: config.ClassicApiType{
									Api: "alerting-profile",
								},
								Scope: "scope",
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"general/alerting-profile/a.json",
			},
		},
		{
			name: "Documents",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/document-dashboard/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "configId1",
					},
					Type:           config.DocumentType{Kind: config.DashboardKind},
					OriginObjectId: "ext-ID-123",
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.NewInMemoryTemplateWithPath("project/document-dashboard/b.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "configId2",
					},
					Type:           config.DocumentType{Kind: config.DashboardKind, Private: true},
					OriginObjectId: "ext-ID-123",
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
				{
					Template: template.NewInMemoryTemplateWithPath("project/document-notebook/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "document",
						ConfigId: "configId3",
					},
					Type:           config.DocumentType{Kind: config.NotebookKind},
					OriginObjectId: "ext-ID-123",
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: true,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"document-dashboard": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId1",
							Config: persistence.ConfigDefinition{
								Name:           "name",
								Parameters:     nil,
								Template:       "a.json",
								OriginObjectId: "ext-ID-123",
								Skip:           true,
							},
							Type: persistence.TypeDefinition{
								Type: config.DocumentType{Kind: config.DashboardKind},
							},
						},
						{
							Id: "configId2",
							Config: persistence.ConfigDefinition{
								Name:           "name",
								Parameters:     nil,
								Template:       "b.json",
								OriginObjectId: "ext-ID-123",
								Skip:           true,
							},
							Type: persistence.TypeDefinition{
								Type: config.DocumentType{Kind: config.DashboardKind, Private: true},
							},
						},
					},
				},
				"document-notebook": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "configId3",
							Config: persistence.ConfigDefinition{
								Name:           "name",
								Parameters:     nil,
								Template:       "a.json",
								OriginObjectId: "ext-ID-123",
								Skip:           true,
							},
							Type: persistence.TypeDefinition{
								Type: config.DocumentType{Kind: config.NotebookKind},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/document-dashboard/a.json",
				"project/document-dashboard/b.json",
				"project/document-notebook/a.json",
			},
		},
		{
			name: "OpenPipeline",
			configs: []config.Config{
				{
					Template: template.NewInMemoryTemplateWithPath("project/openpipeline/a.json", ""),
					Coordinate: coordinate.Coordinate{
						Project:  "project",
						Type:     "openpipeline",
						ConfigId: "bizevents-openpipeline-id",
					},
					Type:           config.OpenPipelineType{Kind: "bizevents"},
					OriginObjectId: "ext-ID-123",
					Parameters: map[string]parameter.Parameter{
						config.NameParameter: &value.ValueParameter{Value: "name"},
					},
					Skip: false,
				},
			},
			expectedConfigs: map[string]persistence.TopLevelDefinition{
				"openpipeline": {
					Configs: []persistence.TopLevelConfigDefinition{
						{
							Id: "bizevents-openpipeline-id",
							Config: persistence.ConfigDefinition{
								Name:           "name",
								Parameters:     nil,
								Template:       "a.json",
								OriginObjectId: "ext-ID-123",
								Skip:           false,
							},
							Type: persistence.TypeDefinition{
								Type: config.OpenPipelineType{Kind: "bizevents"},
							},
						},
					},
				},
			},
			expectedTemplatePaths: []string{
				"project/openpipeline/a.json",
			},
			envVars: map[string]string{
				featureflags.OpenPipeline.EnvName(): "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			fs := testutils.TempFs(t)

			errs := WriteConfigs(&WriterContext{
				Fs:              fs,
				OutputFolder:    "test",
				ProjectFolder:   "project",
				ParametersSerde: config.DefaultParameterParsers,
			}, tc.configs)
			errutils.PrintErrors(errs)
			assert.Equal(t, len(tc.expectedErrs), len(errs), "Produced errors do not match expected errors")

			for i := range tc.expectedErrs {
				assert.ErrorContains(t, errs[i], tc.expectedErrs[i])
			}

			// check all api-folders config file
			for apiType, expectedDefinition := range tc.expectedConfigs {
				content, err := afero.ReadFile(fs, "test/project/"+apiType+"/config.yaml")
				assert.NoError(t, err, "reading config file should not produce an error")

				var actual persistence.TopLevelDefinition
				err = yaml.Unmarshal(content, &actual)
				assert.NoError(t, err, "unmarshalling config file should not produce an error")

				assert.Equal(t, expectedDefinition, actual)
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

func TestNoDuplicateTemplates(t *testing.T) {
	configs := []config.Config{
		{
			Template:   template.NewInMemoryTemplate("template-id ", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "ctype", ConfigId: "cid0"},
			Type:       config.ClassicApiType{},
		},
		{
			Template:   template.NewInMemoryTemplate(" template-id", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "ctype", ConfigId: "cid1"},
			Type:       config.ClassicApiType{},
		},
		{
			Template:   template.NewInMemoryTemplate("template-id", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "ctype", ConfigId: "cid2"},
			Type:       config.ClassicApiType{},
		},
	}
	expectedTemplatePaths := []string{
		"test/project/ctype/template-id.json",
		"test/project/ctype/template-id1.json",
		"test/project/ctype/template-id2.json",
	}

	fs := afero.NewMemMapFs()
	errs := WriteConfigs(&WriterContext{
		Fs:              fs,
		OutputFolder:    "test",
		ProjectFolder:   "project",
		ParametersSerde: config.DefaultParameterParsers,
	}, configs)

	assert.Len(t, errs, 0)

	// check that templates have been created
	for _, path := range expectedTemplatePaths {
		found, err := afero.Exists(fs, path)
		assert.NoError(t, err)
		assert.Equal(t, found, true, "could not find %q", path)
	}
}

func TestOrderedConfigs(t *testing.T) {
	configs := []config.Config{
		{
			Template:   template.NewInMemoryTemplateWithPath("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "b"},
			Type:       config.ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{config.NameParameter: &value.ValueParameter{Value: "name"}},
		},
		{
			Template:   template.NewInMemoryTemplateWithPath("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "a"},
			Type:       config.ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{config.NameParameter: &value.ValueParameter{Value: "name"}},
		},
		{
			Template:   template.NewInMemoryTemplateWithPath("project/alerting-profile/a.json", ""),
			Coordinate: coordinate.Coordinate{Project: "project", Type: "alerting-profile", ConfigId: "c"},
			Type:       config.ClassicApiType{Api: "alerting-profile"},
			Parameters: map[string]parameter.Parameter{config.NameParameter: &value.ValueParameter{Value: "name"}},
		},
	}

	fs := testutils.TempFs(t)

	errs := WriteConfigs(&WriterContext{
		Fs:              fs,
		OutputFolder:    "test",
		ProjectFolder:   "project",
		ParametersSerde: config.DefaultParameterParsers,
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

func TestPrepareFileName(t *testing.T) {
	t.Setenv(environment.MaxFilenameLenKey, "20")
	tests := []struct {
		name          string
		fileExtension string
		expected      string
		expectPanic   bool
	}{
		{"shortname", ".txt", "shortname.txt", false},
		{"verylongfilenameexceedingmaxlength", ".txt", "verylongfilena.txt", false},
		{"special!#", ".txt", "special.txt", false},
		{"namewithlongextension", ".longextension", "name.longextension", false},
		{"namecausingpanic", ".thisextensioniswaytoolong", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("expected panic for input %s, %s", tt.name, tt.fileExtension)
					}
				}()
			}

			result := prepareFileName(tt.name, tt.fileExtension)
			if result != tt.expected && !tt.expectPanic {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
