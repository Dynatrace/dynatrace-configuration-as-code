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

package download

import (
	"fmt"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/google/go-cmp/cmp"
	"gotest.tools/assert"
	"testing"
)

func TestDependencyResolution(t *testing.T) {

	tests := []struct {
		name     string
		setup    project.ConfigsPerType
		expected project.ConfigsPerType
	}{
		{
			"empty works",
			project.ConfigsPerType{},
			project.ConfigsPerType{},
		},
		{
			"single config works",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id", "name", "content"),
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id", "name", "content"),
					},
				},
			},
		},
		{
			"two disjunctive config works",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id", "name", "content"),
					},
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id2", "name2", "content2"),
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id", "name", "content"),
					},
					{
						Type:     config.ClassicApiType{Api: "api-id"},
						Template: template.NewDownloadTemplate("id2", "name2", "content2"),
					},
				},
			},
		},
		{
			"referencing a config works",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", "something something c1-id something something"),
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
					},
					{
						Type:     config.ClassicApiType{Api: "api"},
						Template: template.NewDownloadTemplate("c2-id", "name2", makeTemplateString("something something %s something something", "api", "c1-id")),
						Parameters: config.Parameters{
							createParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
						},
					},
				},
			},
		},
		{
			"cyclic reference works",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", "template of config 1 references config 2: c2-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", "template of config 2 references config 1: c1-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", makeTemplateString("template of config 1 references config 2: %s", "api", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							createParameterName("api", "c2-id"): refParam.New("project", "api", "c2-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", makeTemplateString("template of config 2 references config 1: %s", "api", "c1-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							createParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
						},
					},
				},
			},
		},
		{
			"3-config transitive",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", "template of config 1 references config 2: c2-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", "template of config 2 references config 3: c3-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c3-id", "name3", "template of config 3 references nothing"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", makeTemplateString("template of config 1 references config 2: %s", "api", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							createParameterName("api", "c2-id"): refParam.New("project", "api", "c2-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", makeTemplateString("template of config 2 references config 3: %s", "api", "c3-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							createParameterName("api", "c3-id"): refParam.New("project", "api", "c3-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c3-id", "name3", "template of config 3 references nothing"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
		},
		{
			"3-config transitive over different apis",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", "template of config 1 references config 2: c2-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
				},
				"api-2": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-2"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", "template of config 2 references config 3: c3-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
				},
				"api-3": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-3"},
						Template:   template.NewDownloadTemplate("c3-id", "name3", "template of config 3 references nothing"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-3", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewDownloadTemplate("c1-id", "name", makeTemplateString("template of config 1 references config 2: %s", "api-2", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							createParameterName("api-2", "c2-id"): refParam.New("project", "api-2", "c2-id", "id"),
						},
					},
				},
				"api-2": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-2"},
						Template:   template.NewDownloadTemplate("c2-id", "name2", makeTemplateString("template of config 2 references config 3: %s", "api-3", "c3-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							createParameterName("api-3", "c3-id"): refParam.New("project", "api-3", "c3-id", "id"),
						},
					},
				},
				"api-3": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-3"},
						Template:   template.NewDownloadTemplate("c3-id", "name3", "template of config 3 references nothing"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-3", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
		},
		{
			name: "Scope is replaced in dependency resolution",
			setup: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
				},
			},
			expected: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api", "id2", "id"),
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
				},
			},
		},
		{
			name: "Scope is not replaced if no dependency is present",
			setup: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "HOST-1234"},
						},
					},
				},
			},
			expected: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "HOST-1234"},
						},
					},
				},
			},
		},
		{
			name: "Scope-resolution transitive",
			setup: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id3"},
						},
					},
				},
				"api-2": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id3", "name3", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "id3"},
						Type:       config.SettingsType{SchemaId: "api-2"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "environment"},
						},
					},
				},
			},
			expected: project.ConfigsPerType{
				"api": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id1", "name1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api", "id2", "id"),
						},
					},
					{
						Template:   template.NewDownloadTemplate("id2", "name2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api-2", "id3", "id"),
						},
					},
				},
				"api-2": []config.Config{
					{
						Template:   template.NewDownloadTemplate("id3", "name3", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "id3"},
						Type:       config.SettingsType{SchemaId: "api-2"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "environment"},
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := ResolveDependencies(test.setup)

			assert.DeepEqual(t, result, test.expected, cmp.AllowUnexported(template.DownloadTemplate{}))
		})
	}
}

func makeTemplateString(template, api, configId string) string {
	return fmt.Sprintf(template, "{{."+createParameterName(api, configId)+"}}")
}
