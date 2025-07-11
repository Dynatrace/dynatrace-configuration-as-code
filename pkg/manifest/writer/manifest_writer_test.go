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
	"reflect"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/oauth2/endpoints"
)

func Test_toWriteableProjects(t *testing.T) {
	tests := []struct {
		name          string
		givenProjects map[string]manifest.ProjectDefinition
		wantResult    []persistence.Project
	}{
		{
			name: "creates_simple_projects",
			givenProjects: map[string]manifest.ProjectDefinition{
				"project_a": {
					Name: "a",
					Path: "projects/a",
				},
				"project_b": {
					Name: "b",
					Path: "projects/b",
				},
				"project_c": {
					Name: "c",
					Path: "projects/c",
				},
			},
			wantResult: []persistence.Project{
				{
					Name: "a",
					Path: "projects/a",
				},
				{
					Name: "b",
					Path: "projects/b",
				},
				{
					Name: "c",
					Path: "projects/c",
				},
			},
		},
		{
			"creates_grouping_projects",
			map[string]manifest.ProjectDefinition{
				"project_a": {
					Name: "projects.a",
					Path: "projects/a",
				},
				"project_b": {
					Name: "projects.b",
					Path: "projects/b",
				},
				"project_c": {
					Name: "projects.c",
					Path: "projects/c",
				},
			},
			[]persistence.Project{
				{
					Name: "projects",
					Path: "projects",
					Type: "grouping",
				},
			},
		},
		{
			name: "creates_mixed_projects",
			givenProjects: map[string]manifest.ProjectDefinition{
				"project_a": {
					Name: "projects.a",
					Path: "projects/a",
				},
				"project_b": {
					Name: "projects.b",
					Path: "projects/b",
				},
				"project_c": {
					Name: "projects.c",
					Path: "projects/c",
				},
				"project_alpha": {
					Name: "alpha",
					Path: "special_projects/alpha",
				},
				"nested_project_1": {
					Name: "nested.projects.deeply.grouped.one",
					Path: "nested/projects/deeply/grouped/one",
				},
				"nested_project_2": {
					Name: "nested.projects.deeply.grouped.two",
					Path: "nested/projects/deeply/grouped/two",
				},
			},
			wantResult: []persistence.Project{
				{
					Name: "alpha",
					Path: "special_projects/alpha",
				},
				{
					Name: "nested.projects.deeply.grouped",
					Path: "nested/projects/deeply/grouped",
					Type: "grouping",
				},
				{
					Name: "projects",
					Path: "projects",
					Type: "grouping",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := toWriteableProjects(tt.givenProjects)
			assert.ElementsMatch(t, gotResult, tt.wantResult)
		})
	}
}

func Test_toWriteableEnvironmentGroups(t *testing.T) {
	tests := []struct {
		name       string
		input      manifest.Environments
		wantResult []persistence.Group
	}{
		{
			name: "correctly transforms simple env groups",
			input: manifest.Environments{
				SelectedEnvironments: map[string]manifest.EnvironmentDefinition{
					"env1": {
						Name: "env1",
						URL: manifest.URLDefinition{
							Value: "www.an.Url",
						},
						Group: "group1",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{
								Name: "TokenTest",
							},
						},
					},
					"env2": {
						Name: "env2",
						URL: manifest.URLDefinition{
							Value: "www.an.Url",
						},
						Group: "group1",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{},
							PlatformToken: &manifest.AuthSecret{
								Name:  "token",
								Value: "value",
							},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id-key",
									Value: "client-id-val",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret-key",
									Value: "client-secret-val",
								},
								TokenEndpoint: &manifest.URLDefinition{
									Value: endpoints.Dynatrace.TokenURL,
									Type:  manifest.EnvironmentURLType,
									Name:  "ENV_TOKEN_ENDPOINT",
								},
							},
						},
					},
					"env2a": {
						Name: "env2",
						URL: manifest.URLDefinition{
							Value: "www.an.Url",
						},
						Group: "group1",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id-key",
									Value: "client-id-val",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret-key",
									Value: "client-secret-val",
								},
							},
						},
					},
					"env2b": {
						Name: "env2",
						URL: manifest.URLDefinition{
							Value: "www.an.Url",
						},
						Group: "group1",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{},
							OAuth: &manifest.OAuth{
								ClientID: manifest.AuthSecret{
									Name:  "client-id-key",
									Value: "client-id-val",
								},
								ClientSecret: manifest.AuthSecret{
									Name:  "client-secret-key",
									Value: "client-secret-val",
								},
								TokenEndpoint: &manifest.URLDefinition{
									Value: "http://custom.sso.token.endpoint",
									Type:  manifest.ValueURLType,
								},
							},
						},
					},
					"env3": {
						Name: "env3",
						URL: manifest.URLDefinition{
							Value: "www.an.Url",
						},
						Group: "group2",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{},
						},
					},
				},
			},
			wantResult: []persistence.Group{
				{
					Name: "group1",
					Environments: []persistence.Environment{
						{
							Name: "env1",
							URL:  persistence.TypedValue{Value: "www.an.Url"},
							Auth: persistence.Auth{
								AccessToken: &persistence.AuthSecret{
									Name: "TokenTest",
									Type: "environment",
								},
							},
						},
						{
							Name: "env2",
							URL:  persistence.TypedValue{Value: "www.an.Url"},
							Auth: persistence.Auth{
								AccessToken: &persistence.AuthSecret{
									Name: "",
									Type: "environment",
								},
								PlatformToken: &persistence.AuthSecret{
									Type: persistence.TypeEnvironment,
									Name: "token",
								},
								OAuth: &persistence.OAuth{
									ClientID: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-secret-key",
									},
									TokenEndpoint: &persistence.TypedValue{
										Type:  persistence.TypeEnvironment,
										Value: "ENV_TOKEN_ENDPOINT",
									},
								},
							},
						},
						{
							Name: "env2a",
							URL:  persistence.TypedValue{Value: "www.an.Url"},
							Auth: persistence.Auth{
								AccessToken: &persistence.AuthSecret{
									Name: "",
									Type: "environment",
								},
								OAuth: &persistence.OAuth{
									ClientID: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-secret-key",
									},
								},
							},
						},
						{
							Name: "env2b",
							URL:  persistence.TypedValue{Value: "www.an.Url"},
							Auth: persistence.Auth{
								AccessToken: &persistence.AuthSecret{
									Name: "",
									Type: "environment",
								},
								OAuth: &persistence.OAuth{
									ClientID: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: persistence.AuthSecret{
										Type: persistence.TypeEnvironment,
										Name: "client-secret-key",
									},
									TokenEndpoint: &persistence.TypedValue{
										Value: "http://custom.sso.token.endpoint",
									},
								},
							},
						},
					},
				},
				{
					Name: "group2",
					Environments: []persistence.Environment{
						{
							Name: "env3",
							URL:  persistence.TypedValue{Value: "www.an.Url"},
							Auth: persistence.Auth{
								AccessToken: &persistence.AuthSecret{
									Name: "",
									Type: "environment",
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "returns empty groups for empty env definition",
			input:      manifest.Environments{},
			wantResult: []persistence.Group{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotResult := toWriteableEnvironmentGroups(tt.input); gotResult != nil {
				assert.Equal(t, len(gotResult), len(tt.wantResult))

				// sort Entries sub-slices before checking equality of got and wanted group slices
				for _, g := range gotResult {
					sort.Slice(g.Environments, func(i, j int) bool {
						return g.Environments[i].Name < g.Environments[j].Name
					})
				}

				assert.ElementsMatch(t,
					tt.wantResult,
					gotResult,
				)
			}
		})
	}
}

func Test_toWriteableUrl(t *testing.T) {
	tests := []struct {
		name  string
		input manifest.URLDefinition
		want  persistence.TypedValue
	}{
		{
			"correctly transforms env var Url",
			manifest.URLDefinition{
				Type:  manifest.EnvironmentURLType,
				Name:  "{{ .Env.VARIABLE }}",
				Value: "Some previously resolved value",
			},
			persistence.TypedValue{
				Type:  persistence.TypeEnvironment,
				Value: "{{ .Env.VARIABLE }}",
			},
		},
		{
			"correctly transforms value Url",
			manifest.URLDefinition{
				Type:  manifest.ValueURLType,
				Value: "www.an.Url",
			},
			persistence.TypedValue{
				Value: "www.an.Url",
			},
		},
		{
			"defaults to value Url if no type is defined",
			manifest.URLDefinition{
				Value: "www.an.Url",
			},
			persistence.TypedValue{
				Value: "www.an.Url",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toWriteableURL(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toWriteableURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toWritableToken(t *testing.T) {
	want := &persistence.AuthSecret{
		Name: "VARIABLE",
		Type: "environment",
	}
	got := getAuthSecret(&manifest.AuthSecret{Name: "VARIABLE"})
	assert.Equal(t, want, got)
}

func Test_toWritableNilToken(t *testing.T) {
	got := getAuthSecret(nil)
	assert.Nil(t, got)
}

func Test_toWriteableAccounts(t *testing.T) {

	tests := []struct {
		name  string
		given map[string]manifest.Account
		want  []persistence.Account
	}{
		{
			"simple case",
			map[string]manifest.Account{
				"test": {
					Name:        "test",
					AccountUUID: uuid.MustParse("95a97c92-7137-4f7a-94ff-f29b54b94a72"),
					OAuth: manifest.OAuth{
						ClientID: manifest.AuthSecret{
							Name:  "MY_CLIENT_ID",
							Value: "SECRET!",
						},
						ClientSecret: manifest.AuthSecret{
							Name:  "MY_CLIENT_SECRET",
							Value: "ALSO SECRET!!!",
						},
					},
				},
			},
			[]persistence.Account{
				{
					Name: "test",
					AccountUUID: persistence.TypedValue{
						Value: "95a97c92-7137-4f7a-94ff-f29b54b94a72",
					},
					OAuth: persistence.OAuth{
						ClientID: persistence.AuthSecret{
							Type: persistence.TypeEnvironment,
							Name: "MY_CLIENT_ID",
						},
						ClientSecret: persistence.AuthSecret{
							Type: persistence.TypeEnvironment,
							Name: "MY_CLIENT_SECRET",
						},
					},
				},
			},
		},
		{
			"With optional env var URLs",
			map[string]manifest.Account{
				"test": {
					Name:        "test",
					AccountUUID: uuid.MustParse("95a97c92-7137-4f7a-94ff-f29b54b94a72"),
					ApiUrl: &manifest.URLDefinition{
						Type: manifest.EnvironmentURLType,
						Name: "MY_ENV_URL",
					},
					OAuth: manifest.OAuth{
						ClientID: manifest.AuthSecret{
							Name:  "MY_CLIENT_ID",
							Value: "SECRET!",
						},
						ClientSecret: manifest.AuthSecret{
							Name:  "MY_CLIENT_SECRET",
							Value: "ALSO SECRET!!!",
						},
						TokenEndpoint: &manifest.URLDefinition{
							Type: manifest.EnvironmentURLType,
							Name: "TOKEN_ENDPOINT",
						},
					},
				},
			},
			[]persistence.Account{
				{
					Name: "test",
					AccountUUID: persistence.TypedValue{
						Value: "95a97c92-7137-4f7a-94ff-f29b54b94a72",
					},
					ApiUrl: &persistence.TypedValue{
						Type:  persistence.TypeEnvironment,
						Value: "MY_ENV_URL",
					},
					OAuth: persistence.OAuth{
						ClientID: persistence.AuthSecret{
							Type: persistence.TypeEnvironment,
							Name: "MY_CLIENT_ID",
						},
						ClientSecret: persistence.AuthSecret{
							Type: persistence.TypeEnvironment,
							Name: "MY_CLIENT_SECRET",
						},
						TokenEndpoint: &persistence.TypedValue{
							Type:  persistence.TypeEnvironment,
							Value: "TOKEN_ENDPOINT",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toWriteableAccounts(tt.given)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestWrite(t *testing.T) {
	tests := []struct {
		name          string
		givenManifest manifest.Manifest
		wantJSON      string
	}{
		{
			name: "writes manifest",
			givenManifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					"p1": {
						Name: "p1",
						Path: "projects/p1",
					},
				},
				Environments: manifest.Environments{
					SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
						"env1": {
							Name: "env1",
							URL: manifest.URLDefinition{
								Value: "https://a.dynatrace.environment",
							},
							Group: "group1",
							Auth: manifest.Auth{
								AccessToken: &manifest.AuthSecret{
									Name: "TOKEN_VAR",
								},
							},
						},
					},
					AllEnvironmentNames: map[string]struct{}{
						"env1": {},
					},
					AllGroupNames: map[string]struct{}{
						"group1": {},
					},
				},
			},
			wantJSON: `manifestVersion: "1.0"
projects:
- name: p1
  path: projects/p1
environmentGroups:
- name: group1
  environments:
  - name: env1
    url:
      value: https://a.dynatrace.environment
    auth:
      token:
        type: environment
        name: TOKEN_VAR
`,
		},
		{
			name: "writes manifest with accounts if FF active",
			givenManifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					"p1": {
						Name: "p1",
						Path: "projects/p1",
					},
				},
				Environments: manifest.Environments{
					SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
						"env1": {
							Name: "env1",
							URL: manifest.URLDefinition{
								Value: "https://a.dynatrace.environment",
							},
							Group: "group1",
							Auth: manifest.Auth{
								AccessToken: &manifest.AuthSecret{
									Name: "TOKEN_VAR",
								},
							},
						},
					},
					AllEnvironmentNames: map[string]struct{}{
						"env1": {},
					},
					AllGroupNames: map[string]struct{}{
						"group1": {},
					},
				},
				Accounts: map[string]manifest.Account{
					"account_1": {
						Name:        "account_1",
						AccountUUID: uuid.MustParse("95a97c92-7137-4f7a-94ff-f29b54b94a72"),
						OAuth: manifest.OAuth{
							ClientID: manifest.AuthSecret{
								Name:  "MY_CLIENT_ID",
								Value: "SECRET!",
							},
							ClientSecret: manifest.AuthSecret{
								Name:  "MY_CLIENT_SECRET",
								Value: "ALSO SECRET!!!",
							},
						},
					},
				},
			},
			wantJSON: `manifestVersion: "1.0"
projects:
- name: p1
  path: projects/p1
environmentGroups:
- name: group1
  environments:
  - name: env1
    url:
      value: https://a.dynatrace.environment
    auth:
      token:
        type: environment
        name: TOKEN_VAR
accounts:
- name: account_1
  accountUUID:
    value: 95a97c92-7137-4f7a-94ff-f29b54b94a72
  oAuth:
    clientId:
      type: environment
      name: MY_CLIENT_ID
    clientSecret:
      type: environment
      name: MY_CLIENT_SECRET
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestPath := "manifest.yaml"
			fs := afero.NewMemMapFs()
			err := Write(fs, manifestPath, tt.givenManifest)
			assert.NoError(t, err)

			exists, err := afero.Exists(fs, manifestPath)
			assert.True(t, exists)
			assert.NoError(t, err)
			got, err := afero.ReadFile(fs, manifestPath)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantJSON, string(got))
		})
	}
}
