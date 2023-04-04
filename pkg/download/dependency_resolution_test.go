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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
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
			"referencing a Setting via objectID works",
			project.ConfigsPerType{
				"builtin:some-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:some-setting"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "name", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:some-setting", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "object1-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("869242as-13fw124-f23r24", "name2", "something something object1-objectID something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"builtin:some-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:some-setting"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "name", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:some-setting", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "object1-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("869242as-13fw124-f23r24", "name2", makeTemplateString("something something %s something something", "builtin:some-setting", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: config.Parameters{
							config.ScopeParameter: valueParam.New("environment"),
							createParameterName("builtin:some-setting", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:some-setting", "4fw231-13fw124-f23r24", "id"),
						},
					},
				},
			},
		},
		{
			"referencing a Management Zone Setting via numeric ID works",
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("342342-26re248-w46w48", "mz-with-legacy-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "342342-26re248-w46w48"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRkMGRlZDRhNy1mY2ZlLTQ2MDUtYTEyMy03YWE4ZDBmYTVhMja-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", "something something -4292415658385853785 something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("869242as-13fw124-f23r24", "references-legacy-uuid-mz", "something something 3277109782074005416 something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object1-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("342342-26re248-w46w48", "mz-with-legacy-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "342342-26re248-w46w48"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRkMGRlZDRhNy1mY2ZlLTQ2MDUtYTEyMy03YWE4ZDBmYTVhMja-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", makeTemplateString("something something %s something something", "builtin:management-zones", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							createParameterName("builtin:management-zones", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:management-zones", "4fw231-13fw124-f23r24", "id"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("869242as-13fw124-f23r24", "references-legacy-uuid-mz", makeTemplateString("something something %s something something", "builtin:management-zones", "342342-26re248-w46w48")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object1-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							createParameterName("builtin:management-zones", "342342-26re248-w46w48"): refParam.New("project", "builtin:management-zones", "342342-26re248-w46w48", "id"),
						},
					},
				},
			},
		},
		{
			"resolution does not break if getting a management zone numeric ID fails",
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "OBJECT ID THAT CAN NOT BE PARSED INTO A NUMERIC ID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", "something something -4292415658385853785 something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "OBJECT ID THAT CAN NOT BE PARSED INTO A NUMERIC ID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", "something something -4292415658385853785 something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
			},
		},
		{
			"management zone references could be resolved by object ID as well",
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "mz-object-id",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", "something something mz-object-id something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
			},
			project.ConfigsPerType{
				"builtin:management-zones": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewDownloadTemplate("4fw231-13fw124-f23r24", "mz-with-new-uuid", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "mz-object-id",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
				},
				"builtin:other-setting": []config.Config{
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewDownloadTemplate("5242as-13fw124-f23r24", "references-new-uuid-mz", makeTemplateString("something something %s something something", "builtin:management-zones", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							createParameterName("builtin:management-zones", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:management-zones", "4fw231-13fw124-f23r24", "id"),
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
