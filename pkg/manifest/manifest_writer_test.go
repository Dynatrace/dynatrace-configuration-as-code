//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
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

package manifest

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/oauth2/endpoints"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	"reflect"
	"sort"
	"testing"
)

func Test_toWriteableProjects(t *testing.T) {
	tests := []struct {
		name          string
		givenProjects map[string]ProjectDefinition
		wantResult    []project
	}{
		{
			name: "creates_simple_projects",
			givenProjects: map[string]ProjectDefinition{
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
			wantResult: []project{
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
			map[string]ProjectDefinition{
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
			[]project{
				{
					Name: "projects",
					Path: "projects",
					Type: "grouping",
				},
			},
		},
		{
			name: "creates_mixed_projects",
			givenProjects: map[string]ProjectDefinition{
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
			wantResult: []project{
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
			assert.DeepEqual(t, gotResult, tt.wantResult, cmpopts.SortSlices(func(a, b project) bool { return a.Name < b.Name }))
		})
	}
}

func Test_toWriteableEnvironmentGroups(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]EnvironmentDefinition
		wantResult []group
	}{
		{
			name: "correctly transforms simple env groups",
			input: map[string]EnvironmentDefinition{
				"env1": {
					Name: "env1",
					Type: Classic,
					URL: URLDefinition{
						Value: "www.an.Url",
					},
					Group: "group1",
					Auth: Auth{
						Token: AuthSecret{
							Name: "TokenTest",
						},
					},
				},
				"env2": {
					Name: "env2",
					Type: Platform,
					URL: URLDefinition{
						Value: "www.an.Url",
					},
					Group: "group1",
					Auth: Auth{
						Token: AuthSecret{},
						OAuth: OAuth{
							ClientID: AuthSecret{
								Name:  "client-id-key",
								Value: "client-id-val",
							},
							ClientSecret: AuthSecret{
								Name:  "client-secret-key",
								Value: "client-secret-val",
							},
							TokenEndpoint: &URLDefinition{
								Value: endpoints.Dynatrace.TokenURL,
								Type:  EnvironmentURLType,
								Name:  "ENV_TOKEN_ENDPOINT",
							},
						},
					},
				},
				"env2a": {
					Name: "env2",
					Type: Platform,
					URL: URLDefinition{
						Value: "www.an.Url",
					},
					Group: "group1",
					Auth: Auth{
						Token: AuthSecret{},
						OAuth: OAuth{
							ClientID: AuthSecret{
								Name:  "client-id-key",
								Value: "client-id-val",
							},
							ClientSecret: AuthSecret{
								Name:  "client-secret-key",
								Value: "client-secret-val",
							},
						},
					},
				},
				"env2b": {
					Name: "env2",
					Type: Platform,
					URL: URLDefinition{
						Value: "www.an.Url",
					},
					Group: "group1",
					Auth: Auth{
						Token: AuthSecret{},
						OAuth: OAuth{
							ClientID: AuthSecret{
								Name:  "client-id-key",
								Value: "client-id-val",
							},
							ClientSecret: AuthSecret{
								Name:  "client-secret-key",
								Value: "client-secret-val",
							},
							TokenEndpoint: &URLDefinition{
								Value: "http://custom.sso.token.endpoint",
								Type:  ValueURLType,
							},
						},
					},
				},
				"env3": {
					Name: "env3",
					Type: Classic,
					URL: URLDefinition{
						Value: "www.an.Url",
					},
					Group: "group2",
					Auth: Auth{
						Token: AuthSecret{},
					},
				},
			},
			wantResult: []group{
				{
					Name: "group1",
					Environments: []environment{
						{
							Name: "env1",
							URL:  url{Value: "www.an.Url"},
							Auth: auth{
								Token: authSecret{
									Name: "TokenTest",
									Type: "environment",
								},
							},
						},
						{
							Name: "env2",
							URL:  url{Value: "www.an.Url"},
							Auth: auth{
								Token: authSecret{
									Name: "env2_TOKEN",
									Type: "environment",
								},
								OAuth: &oAuth{
									ClientID: authSecret{
										Type: typeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: authSecret{
										Type: typeEnvironment,
										Name: "client-secret-key",
									},
									TokenEndpoint: &url{
										Type:  urlTypeEnvironment,
										Value: "ENV_TOKEN_ENDPOINT",
									},
								},
							},
						},
						{
							Name: "env2a",
							URL:  url{Value: "www.an.Url"},
							Auth: auth{
								Token: authSecret{
									Name: "env2_TOKEN",
									Type: "environment",
								},
								OAuth: &oAuth{
									ClientID: authSecret{
										Type: typeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: authSecret{
										Type: typeEnvironment,
										Name: "client-secret-key",
									},
								},
							},
						},
						{
							Name: "env2b",
							URL:  url{Value: "www.an.Url"},
							Auth: auth{
								Token: authSecret{
									Name: "env2_TOKEN",
									Type: "environment",
								},
								OAuth: &oAuth{
									ClientID: authSecret{
										Type: typeEnvironment,
										Name: "client-id-key",
									},
									ClientSecret: authSecret{
										Type: typeEnvironment,
										Name: "client-secret-key",
									},
									TokenEndpoint: &url{
										Value: "http://custom.sso.token.endpoint",
									},
								},
							},
						},
					},
				},
				{
					Name: "group2",
					Environments: []environment{
						{
							Name: "env3",
							URL:  url{Value: "www.an.Url"},
							Auth: auth{
								Token: authSecret{
									Name: "env3_TOKEN",
									Type: "environment",
								},
							},
						},
					},
				},
			},
		},
		{
			"returns empty groups for empty env defintion",
			map[string]EnvironmentDefinition{},
			[]group{},
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

				assert.DeepEqual(t,
					tt.wantResult,
					gotResult,
					cmpopts.SortSlices(func(a, b group) bool { return a.Name < b.Name }),
				)
			}
		})
	}
}

func Test_toWriteableUrl(t *testing.T) {
	tests := []struct {
		name  string
		input EnvironmentDefinition
		want  url
	}{
		{
			"correctly transforms env var Url",
			EnvironmentDefinition{
				Name: "NAME",
				URL: URLDefinition{
					Type:  EnvironmentURLType,
					Name:  "{{ .Env.VARIABLE }}",
					Value: "Some previously resolved value",
				},
				Group: "GROUP",
				Auth: Auth{
					Token: AuthSecret{},
				},
			},
			url{
				Type:  urlTypeEnvironment,
				Value: "{{ .Env.VARIABLE }}",
			},
		},
		{
			"correctly transforms value Url",
			EnvironmentDefinition{
				Name: "NAME",
				URL: URLDefinition{
					Type:  ValueURLType,
					Value: "www.an.Url",
				},
				Group: "GROUP",
				Auth: Auth{
					Token: AuthSecret{},
				},
			},
			url{
				Value: "www.an.Url",
			},
		},
		{
			"defaults to value Url if no type is defined",
			EnvironmentDefinition{
				Name: "NAME",
				URL: URLDefinition{
					Value: "www.an.Url",
				},
				Group: "GROUP",
				Auth: Auth{
					Token: AuthSecret{},
				},
			},
			url{
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
	tests := []struct {
		name  string
		input EnvironmentDefinition
		want  authSecret
	}{
		{
			"correctly transforms env var token",
			EnvironmentDefinition{
				Name:  "NAME",
				URL:   URLDefinition{},
				Group: "GROUP",
				Auth: Auth{
					Token: AuthSecret{Name: "VARIABLE"},
				},
			},
			authSecret{
				Name: "VARIABLE",
				Type: "environment",
			},
		},
		{
			"defaults to assumed token name if nothing is defined",
			EnvironmentDefinition{
				Name:  "NAME",
				URL:   URLDefinition{},
				Group: "GROUP",

				Auth: Auth{
					Token: AuthSecret{},
				},
			},
			authSecret{
				Name: "NAME_TOKEN",
				Type: "environment",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTokenSecret(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getTokenSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
