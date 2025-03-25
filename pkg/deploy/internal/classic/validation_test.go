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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/testutils"
)

func TestValidate_NoErrorForNonClassicAPIs(t *testing.T) {
	validator := NewValidator()
	err := validator.Validate(newTestConfigForValidation(t,
		coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "abcde"},
		config.SettingsType{SchemaId: "builtin:management-zones", SchemaVersion: "1.2.3"},
		map[string]parameter.Parameter{}))

	assert.NoError(t, err)
}

func TestValidate_NoErrorForNonUniqueNames(t *testing.T) {
	validator := NewValidator()
	err := validator.Validate(newTestConfigForValidation(t,
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

	validator := NewValidator()

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsWeb, parameters))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsWeb, parameters))
	assert.NoError(t, err2)
}

func TestValidate_NoErrorForConfigsWithDifferentNames(t *testing.T) {
	validator := NewValidator()

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name1")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name2")}))
	assert.NoError(t, err2)
}

func TestValidate_ErrorForConfigWithSameName(t *testing.T) {
	validator := NewValidator()

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.ApplicationMobile, map[string]parameter.Parameter{
		config.NameParameter: value.New("name")}))
	assert.Error(t, err2)
}

func TestValidate_NoErrorForSameNameInDifferentScopes(t *testing.T) {
	validator := NewValidator()

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
	validator := NewValidator()

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
	validator := NewValidator()

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}))
	assert.Error(t, err2)
}

func TestValidate_ErrorForSameNameAndScopeWithDifferentScopeCheckedEarlier(t *testing.T) {
	validator := NewValidator()

	err1 := validator.Validate(newTestClassicConfigForValidation(t, "config1", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope")}))
	assert.NoError(t, err1)

	err2 := validator.Validate(newTestClassicConfigForValidation(t, "config2", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope1")}))
	assert.NoError(t, err2)

	err3 := validator.Validate(newTestClassicConfigForValidation(t, "config3", api.KeyUserActionsMobile, map[string]parameter.Parameter{
		config.NameParameter:  value.New("name"),
		config.ScopeParameter: value.New("scope1")}))

	assert.Error(t, err3)
}

func TestValidate_ValidateCompoundParameterName(t *testing.T) {

	t.Run("compound resolves to different values - no error", func(t *testing.T) {
		validator := NewValidator()
		compoundParam1, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("jenny"),
				"lastName":           value.New("curran"),
				config.NameParameter: compoundParam1,
			}}

		compoundParam2, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam2,
			}}

		err1 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c2)
		assert.NoError(t, err2)
	})

	t.Run("compound parameters with references to different config - no error", func(t *testing.T) {
		validator := NewValidator()
		compoundParam1, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          reference.New("project", api.ApplicationMobile, "SECOND", "lastName"),
				"lastName":           value.New("curran"),
				config.NameParameter: compoundParam1,
			}}

		compoundParam2, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          reference.New("project", api.ApplicationMobile, "FIRST", "lastName"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam2,
			}}

		err1 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c2)
		assert.NoError(t, err2)
	})

	t.Run("compound parameters with references to same config - error", func(t *testing.T) {
		t.Skipf("This should produce an error, but current quickfix misses that these resolve to the same value")
		validator := NewValidator()
		compoundParam1, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          reference.New("project", api.ApplicationMobile, "ZERO", "firstName"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam1,
			}}
		//compound value == "forrest gump"

		compoundParam2, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          reference.New("project", api.ApplicationMobile, "ZERO", "firstName"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam2,
			}}
		//compound value == "forrest gump"
		// names equal -> error

		err1 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c2)
		assert.Error(t, err2)
	})

	t.Run("compound resolves to same name - error", func(t *testing.T) {
		validator := NewValidator()
		compoundParam1, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam1,
			}}

		compoundParam2, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: compoundParam2,
			}}

		err1 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c2)
		assert.Error(t, err2)
	})

	t.Run("compound resolves to same name - error", func(t *testing.T) {
		validator := NewValidator()

		c0 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "ZERO", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: value.New("forrest gump"),
			}}

		compoundParam1, _ := compound.New("name", "{{ .firstName }} {{ .lastName }}", []parameter.ParameterReference{
			{Config: coordinate.Coordinate{ConfigId: "ZERO", Project: "project", Type: api.ApplicationMobile}, Property: "firstName"},
			{Config: coordinate.Coordinate{ConfigId: "ZERO", Project: "project", Type: api.ApplicationMobile}, Property: "lastName"},
		})

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: compoundParam1}}

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: compoundParam1}}

		err1 := validator.Validate([]project.Project{}, c0)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err2)

		err3 := validator.Validate([]project.Project{}, c2)
		assert.Error(t, err3)

	})

	t.Run("reference to same config - error", func(t *testing.T) {
		validator := NewValidator()

		c0 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "ZERO", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: value.New("forrest gump"),
			}}

		ref1 := reference.New("project", api.ApplicationMobile, "ZERO", "firstName")

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: ref1}}

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: ref1}}

		err1 := validator.Validate([]project.Project{}, c0)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err2)

		err3 := validator.Validate([]project.Project{}, c2)
		assert.Error(t, err3)

	})

	t.Run("reference to same config with different property - no error", func(t *testing.T) {
		validator := NewValidator()

		c0 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "ZERO", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{
				"firstName":          value.New("forrest"),
				"lastName":           value.New("gump"),
				config.NameParameter: value.New("forrest gump"),
			}}

		ref1 := reference.New("project", api.ApplicationMobile, "ZERO", "firstName")
		ref2 := reference.New("project", api.ApplicationMobile, "ZERO", "lastName")

		c1 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "FIRST", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: ref1}}

		c2 := config.Config{
			Type:       config.ClassicApiType{Api: api.ApplicationMobile},
			Coordinate: coordinate.Coordinate{ConfigId: "SECOND", Project: "project", Type: api.ApplicationMobile},
			Parameters: map[string]parameter.Parameter{config.NameParameter: ref2}}

		err1 := validator.Validate([]project.Project{}, c0)
		assert.NoError(t, err1)

		err2 := validator.Validate([]project.Project{}, c1)
		assert.NoError(t, err2)

		err3 := validator.Validate([]project.Project{}, c2)
		assert.NoError(t, err3)

	})

}

func newTestConfigForValidation(t *testing.T, coordinate coordinate.Coordinate, configType config.Type, parameters map[string]parameter.Parameter) ([]project.Project, config.Config) {
	return []project.Project{}, config.Config{
		Coordinate:  coordinate,
		Type:        configType,
		Environment: "dev",
		Parameters:  parameters,
		Template:    testutils.GenerateDummyTemplate(t),
	}
}

func newTestClassicConfigForValidation(t *testing.T, configId string, apiID string, parameters map[string]parameter.Parameter) ([]project.Project, config.Config) {
	return newTestConfigForValidation(
		t,
		coordinate.Coordinate{Project: "project", Type: apiID, ConfigId: configId},
		config.ClassicApiType{Api: apiID}, parameters)
}
