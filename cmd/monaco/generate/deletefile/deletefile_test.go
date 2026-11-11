//go:build unit

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package deletefile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

func newCredentialVaultConfig(nameParam parameter.Parameter) config.Config {
	params := config.Parameters{}
	if nameParam != nil {
		params[config.NameParameter] = nameParam
	}

	return config.Config{
		Coordinate: coordinate.Coordinate{Project: "project", Type: string(config.CredentialVaultID), ConfigId: "credential"},
		Type:       config.CredentialVault{},
		Parameters: params,
	}
}

func TestCreateDeleteEntry_CredentialVault_IdentifyByName(t *testing.T) {
	entry, err := createDeleteEntry(newCredentialVaultConfig(value.New("My credential")), api.NewAPIs(), project.Project{})
	require.NoError(t, err)

	// project and id must not be set, as the delete file loader rejects them for name-identified types
	assert.Equal(t, persistence.DeleteEntry{
		Type:       string(config.CredentialVaultID),
		ConfigName: "My credential",
	}, entry)
}

func TestCreatDeleteEntry_CredentialVault_Errors(t *testing.T) {
	invalidNames := []struct {
		name      string
		nameParam parameter.Parameter
	}{
		{name: "missing name", nameParam: nil},
		{name: "empty name", nameParam: value.New("")},
		{name: "non-string name", nameParam: value.New(42)},
		{name: "reference name", nameParam: reference.New("project", "credential-vault-v2", "other", "name")},
	}

	for _, tt := range invalidNames {
		t.Run(tt.name+" fails", func(t *testing.T) {
			_, err := createDeleteEntry(newCredentialVaultConfig(tt.nameParam), api.NewAPIs(), project.Project{})
			assert.Error(t, err)
		})
	}
}

const testEnv = "env"

const keyUserActionTemplate = `{"name": "{{ .name }}", "domain": "{{ .domain }}", "actionType": "{{ .actionType }}"}`

// newClassicAPIConfig builds a classic API config in testEnv.
func newClassicAPIConfig(apiID string, params config.Parameters, tmpl string) config.Config {
	return config.Config{
		Coordinate:  coordinate.Coordinate{Project: "project", Type: apiID, ConfigId: "child"},
		Type:        config.ClassicApiType{Api: apiID},
		Template:    template.NewInMemoryTemplate("child", tmpl),
		Parameters:  params,
		Environment: testEnv,
	}
}

// scopeReference references the parent application-web config "app" created by newProjectWithParent.
func scopeReference() parameter.Parameter {
	return reference.New("project", api.ApplicationWeb, "app", "id")
}

// newProjectWithParent returns a project containing the parent config "app" in testEnv.
func newProjectWithParent(parentParams config.Parameters) project.Project {
	return project.Project{
		Id: "project",
		Configs: project.ConfigsPerTypePerEnvironments{
			testEnv: project.ConfigsPerType{
				api.ApplicationWeb: []config.Config{
					{
						Coordinate:  coordinate.Coordinate{Project: "project", Type: api.ApplicationWeb, ConfigId: "app"},
						Type:        config.ClassicApiType{Api: api.ApplicationWeb},
						Parameters:  parentParams,
						Environment: testEnv,
					},
				},
			},
		},
	}
}

func TestGetScopeFromEntry(t *testing.T) {
	validProject := newProjectWithParent(config.Parameters{config.NameParameter: value.New("My App")})

	t.Run("returns name of referenced scope config", func(t *testing.T) {
		c := newClassicAPIConfig(api.KeyUserActionsMobile, config.Parameters{config.ScopeParameter: scopeReference()}, "{}")

		scope, err := getScopeFromEntry(c, validProject)

		require.NoError(t, err)
		assert.Equal(t, "My App", scope)
	})

	errorCases := []struct {
		name    string
		params  config.Parameters
		project project.Project
		env     string
	}{
		{
			name:    "missing scope parameter",
			params:  config.Parameters{},
			project: validProject,
			env:     testEnv,
		},
		{
			name:    "scope parameter without references",
			params:  config.Parameters{config.ScopeParameter: value.New("APPLICATION-1234")},
			project: validProject,
			env:     testEnv,
		},
		{
			name:    "referenced config does not exist",
			params:  config.Parameters{config.ScopeParameter: reference.New("project", api.ApplicationWeb, "unknown", "id")},
			project: validProject,
			env:     testEnv,
		},
		{
			name:    "referenced config exists only in other environment",
			params:  config.Parameters{config.ScopeParameter: scopeReference()},
			project: validProject,
			env:     "other-env",
		},
		{
			name:    "referenced config has no name parameter",
			params:  config.Parameters{config.ScopeParameter: scopeReference()},
			project: newProjectWithParent(config.Parameters{}),
			env:     testEnv,
		},
		{
			name:    "name of referenced config is no value parameter",
			params:  config.Parameters{config.ScopeParameter: scopeReference()},
			project: newProjectWithParent(config.Parameters{config.NameParameter: environment.New("APP_NAME")}),
			env:     testEnv,
		},
		{
			name:    "name of referenced config is no string",
			params:  config.Parameters{config.ScopeParameter: scopeReference()},
			project: newProjectWithParent(config.Parameters{config.NameParameter: value.New(42)}),
			env:     testEnv,
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.name+" fails", func(t *testing.T) {
			c := newClassicAPIConfig(api.KeyUserActionsMobile, tt.params, "{}")
			c.Environment = tt.env

			scope, err := getScopeFromEntry(c, tt.project)

			assert.Error(t, err)
			assert.Empty(t, scope)
		})
	}
}

func TestGetCustomValuesFromEntry(t *testing.T) {
	apis := api.NewAPIs()

	t.Run("returns nil for types other than key-user-actions-web", func(t *testing.T) {
		c := newClassicAPIConfig(api.KeyUserActionsMobile, config.Parameters{config.ScopeParameter: scopeReference()}, keyUserActionTemplate)

		values, err := getCustomValuesFromEntry(c, apis)

		require.NoError(t, err)
		assert.Nil(t, values)
	})

	t.Run("returns domain and actionType of key-user-actions-web", func(t *testing.T) {
		// the scope reference can't be resolved offline and must be dropped instead of failing resolution
		c := newClassicAPIConfig(api.KeyUserActionsWeb, config.Parameters{
			config.NameParameter:  value.New("action"),
			config.ScopeParameter: scopeReference(),
			"domain":              value.New("example.com"),
			"actionType":          value.New("Load"),
		}, keyUserActionTemplate)

		values, err := getCustomValuesFromEntry(c, apis)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"domain": "example.com", "actionType": "Load"}, values)
		assert.NotContains(t, c.Parameters, config.ScopeParameter, "reference parameters are removed from the config")
	})

	t.Run("returns empty values if rendered JSON lacks domain and actionType", func(t *testing.T) {
		c := newClassicAPIConfig(api.KeyUserActionsWeb, config.Parameters{config.NameParameter: value.New("action")}, `{"name": "{{ .name }}"}`)

		values, err := getCustomValuesFromEntry(c, apis)

		require.NoError(t, err)
		assert.Equal(t, map[string]string{"domain": "", "actionType": ""}, values)
	})

	errorCases := []struct {
		name     string
		params   config.Parameters
		template string
	}{
		{
			name:     "unresolvable parameter",
			params:   config.Parameters{"domain": environment.New("MONACO_TEST_UNSET_ENV_VAR_FOR_DELETEFILE")},
			template: keyUserActionTemplate,
		},
		{
			name:     "invalid template",
			params:   config.Parameters{config.NameParameter: value.New("action")},
			template: `{"name": "{{ .name }"}`,
		},
		{
			name:     "rendered template is no JSON",
			params:   config.Parameters{config.NameParameter: value.New("action")},
			template: `not json`,
		},
		{
			name:     "rendered JSON contains non-string values",
			params:   config.Parameters{config.NameParameter: value.New("action")},
			template: `{"name": "{{ .name }}", "domain": 42}`,
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.name+" fails", func(t *testing.T) {
			c := newClassicAPIConfig(api.KeyUserActionsWeb, tt.params, tt.template)

			values, err := getCustomValuesFromEntry(c, apis)

			assert.Error(t, err)
			assert.Nil(t, values)
		})
	}
}

func TestCreateConfigAPIEntry(t *testing.T) {
	apis := api.NewAPIs()
	p := newProjectWithParent(config.Parameters{config.NameParameter: value.New("My App")})

	t.Run("API without parent only sets type and name", func(t *testing.T) {
		c := newClassicAPIConfig(api.ApplicationWeb, config.Parameters{config.NameParameter: value.New("app")}, "{}")

		entry, err := createConfigAPIEntry(c, apis, p)

		require.NoError(t, err)
		assert.Equal(t, persistence.DeleteEntry{
			Type:         api.ApplicationWeb,
			ConfigName:   "app",
			CustomValues: map[string]string{},
		}, entry)
	})

	t.Run("API with parent sets scope", func(t *testing.T) {
		c := newClassicAPIConfig(api.KeyUserActionsMobile, config.Parameters{
			config.NameParameter:  value.New("action"),
			config.ScopeParameter: scopeReference(),
		}, "{}")

		entry, err := createConfigAPIEntry(c, apis, p)

		require.NoError(t, err)
		assert.Equal(t, persistence.DeleteEntry{
			Type:       api.KeyUserActionsMobile,
			ConfigName: "action",
			Scope:      "My App",
		}, entry)
	})

	t.Run("key-user-actions-web sets scope and custom values", func(t *testing.T) {
		c := newClassicAPIConfig(api.KeyUserActionsWeb, config.Parameters{
			config.NameParameter:  value.New("action"),
			config.ScopeParameter: scopeReference(),
			"domain":              value.New("example.com"),
			"actionType":          value.New("Load"),
		}, keyUserActionTemplate)

		entry, err := createConfigAPIEntry(c, apis, p)

		require.NoError(t, err)
		assert.Equal(t, persistence.DeleteEntry{
			Type:         api.KeyUserActionsWeb,
			ConfigName:   "action",
			Scope:        "My App",
			CustomValues: map[string]string{"domain": "example.com", "actionType": "Load"},
		}, entry)
	})

	errorCases := []struct {
		name     string
		apiID    string
		params   config.Parameters
		template string
	}{
		{
			name:     "missing name",
			apiID:    api.ApplicationWeb,
			params:   config.Parameters{},
			template: "{}",
		},
		{
			name:     "missing scope for API with parent",
			apiID:    api.KeyUserActionsMobile,
			params:   config.Parameters{config.NameParameter: value.New("action")},
			template: "{}",
		},
		{
			name:  "invalid custom values for key-user-actions-web",
			apiID: api.KeyUserActionsWeb,
			params: config.Parameters{
				config.NameParameter:  value.New("action"),
				config.ScopeParameter: scopeReference(),
			},
			template: "not json",
		},
	}

	for _, tt := range errorCases {
		t.Run(tt.name+" fails", func(t *testing.T) {
			c := newClassicAPIConfig(tt.apiID, tt.params, tt.template)

			entry, err := createConfigAPIEntry(c, apis, p)

			assert.Error(t, err)
			assert.Equal(t, persistence.DeleteEntry{}, entry)
		})
	}
}
