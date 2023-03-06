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
			"creates_simple_projects",
			map[string]ProjectDefinition{
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
			[]project{
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
			"creates_mixed_projects",
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
			[]project{
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
			"correctly transforms simple env groups",
			map[string]EnvironmentDefinition{
				"env1": {
					Name: "env1",
					Type: Classic,
					url: UrlDefinition{
						Value: "www.an.url",
					},
					Group: "group1",
					Token: Token{
						Name: "TokenTest",
					},
				},
				"env2": {
					Name: "env2",
					Type: Platform,
					url: UrlDefinition{
						Value: "www.an.url",
					},
					Group: "group1",
					Token: Token{},
				},
				"env3": {
					Name: "env3",
					Type: Classic,
					url: UrlDefinition{
						Value: "www.an.url",
					},
					Group: "group2",
					Token: Token{},
				},
			},
			[]group{
				{
					Name: "group1",
					Environments: []environment{
						{
							Name: "env1",
							Type: "classic",
							Url:  url{Value: "www.an.url"},
							Token: tokenConfig{
								Name: "TokenTest",
								Type: "environment",
							},
						},
						{
							Name: "env2",
							Type: "platform",
							Url:  url{Value: "www.an.url"},
							Token: tokenConfig{
								Name: "env2_TOKEN",
								Type: "environment",
							},
						},
					},
				},
				{
					Name: "group2",
					Environments: []environment{
						{
							Name: "env3",
							Type: "classic",
							Url:  url{Value: "www.an.url"},
							Token: tokenConfig{
								Name: "env3_TOKEN",
								Type: "environment",
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
					gotResult,
					tt.wantResult,
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
			"correctly transforms env var url",
			EnvironmentDefinition{
				Name: "NAME",
				url: UrlDefinition{
					Type:  EnvironmentUrlType,
					Value: "{{ .Env.VARIABLE }}",
				},
				Group: "GROUP",
				Token: Token{},
			},
			url{
				Type:  "environment",
				Value: "{{ .Env.VARIABLE }}",
			},
		},
		{
			"correctly transforms value url",
			EnvironmentDefinition{
				Name: "NAME",
				url: UrlDefinition{
					Type:  ValueUrlType,
					Value: "www.an.url",
				},
				Group: "GROUP",
				Token: Token{},
			},
			url{
				Value: "www.an.url",
			},
		},
		{
			"defaults to value url if no type is defined",
			EnvironmentDefinition{
				Name: "NAME",
				url: UrlDefinition{
					Value: "www.an.url",
				},
				Group: "GROUP",
				Token: Token{},
			},
			url{
				Value: "www.an.url",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toWriteableUrl(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toWriteableUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toWritableToken(t *testing.T) {
	tests := []struct {
		name  string
		input EnvironmentDefinition
		want  tokenConfig
	}{
		{
			"correctly transforms env var token",
			EnvironmentDefinition{
				Name:  "NAME",
				url:   UrlDefinition{},
				Group: "GROUP",
				Token: Token{Name: "VARIABLE"},
			},
			tokenConfig{
				Name: "VARIABLE",
				Type: "environment",
			},
		},
		{
			"defaults to assumed token name if nothing is defined",
			EnvironmentDefinition{
				Name:  "NAME",
				url:   UrlDefinition{},
				Group: "GROUP",
				Token: Token{},
			},
			tokenConfig{
				Name: "NAME_TOKEN",
				Type: "environment",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toWritableToken(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toWritableToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
