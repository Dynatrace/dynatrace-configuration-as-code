//go:build unit

/**
 * @license
 * Copyright 2024 Dynatrace LLC
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

package classic

import (
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestValidate_NoErrorForNonClassicAPIs(t *testing.T) {
	validator := &Validator{}
	err := validator.Validate(
		newTestConfigForValidation(t,
			coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "abcde"},
			config.SettingsType{SchemaId: "builtin:management-zones", SchemaVersion: "1.2.3"},
			map[string]parameter.Parameter{}))

	assert.NoError(t, err)
}

func TestValidate_NoErrorForNonUniqueNames(t *testing.T) {
	validator := &Validator{}
	err := validator.Validate(
		newTestConfigForValidation(t,
			coordinate.Coordinate{
				Project:  "project",
				Type:     api.Dashboard,
				ConfigId: "sampleDashboard",
			},
			config.ClassicApiType{Api: api.Dashboard},
			map[string]parameter.Parameter{}))

	assert.NoError(t, err)
}

func TestValidate_NoValidationPerformedForKeyUserActionsWeb(t *testing.T) {
	parameters := map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}

	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsWeb, parameters))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsWeb, parameters))
	assert.NoError(t, err2)
}

func TestValidate_NoErrorForConfigsWithDifferentNames(t *testing.T) {
	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name1")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name2")}))
	assert.NoError(t, err2)
}

func TestValidate_ErrorForConfigWithSameName(t *testing.T) {
	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name")}))
	assert.Error(t, err2)
}

func TestValidate_NoErrorForSameNameInDifferentScopes(t *testing.T) {
	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope1")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope2")}))
	assert.NoError(t, err2)
}

func TestValidate_NoErrorForDifferentNamesInDifferentScopes(t *testing.T) {
	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name1"),
		config.ScopeParameter: value.New("scope1")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name2"),
		config.ScopeParameter: value.New("scope2")}))
	assert.NoError(t, err2)
}

func TestValidate_ErrorForSameNameAndScope(t *testing.T) {
	validator := &Validator{}

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}))
	assert.Error(t, err2)
}

func newTestConfigForValidation(t *testing.T, coordinate coordinate.Coordinate, configType config.Type, parameters map[string]parameter.Parameter) config.Config {
	return config.Config{
		Coordinate:  coordinate,
		Type:        configType,
		Environment: "dev",
		Parameters:  parameters,
		Template:    testutils.GenerateDummyTemplate(t),
	}
}

func newTestClassicConfigForValidation(t *testing.T, configId string, apiID string, parameters map[string]parameter.Parameter) config.Config {
	return newTestConfigForValidation(
		t,
		coordinate.Coordinate{Project: "project", Type: apiID, ConfigId: configId},
		config.ClassicApiType{Api: apiID}, parameters)
}
