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

package dependency_resolution

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	refParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution/resolver"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
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
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
					},
				},
			},
		},
		{
			"two disjunctive config works",
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id2", "content2"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id"},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("id2", "content2"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
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
						Template:   template.NewInMemoryTemplate("c1-id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", "something something c1-id something something"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c1-id", "content"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString("something something %s something something", "api", "c1-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("869242as-13fw124-f23r24", "something something object1-objectID something something"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("869242as-13fw124-f23r24", makeTemplateString("something something %s something something", "builtin:some-setting", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: config.Parameters{
							config.ScopeParameter: valueParam.New("environment"),
							resolver.CreateParameterName("builtin:some-setting", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:some-setting", "4fw231-13fw124-f23r24", "id"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewInMemoryTemplate("342342-26re248-w46w48", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", "something something -4292415658385853785 something something"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewInMemoryTemplate("869242as-13fw124-f23r24", "something something 3277109782074005416 something something"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:management-zones", ConfigId: "4fw231-13fw124-f23r24"},
						OriginObjectId: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:management-zones"},
						Template:       template.NewInMemoryTemplate("342342-26re248-w46w48", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", makeTemplateString("something something %s something something", "builtin:management-zones", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							resolver.CreateParameterName("builtin:management-zones", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:management-zones", "4fw231-13fw124-f23r24", "id"),
						},
					},
					{
						Type:           config.SettingsType{SchemaId: "builtin:other-setting"},
						Template:       template.NewInMemoryTemplate("869242as-13fw124-f23r24", makeTemplateString("something something %s something something", "builtin:management-zones", "342342-26re248-w46w48")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "869242as-13fw124-f23r24"},
						OriginObjectId: "object1-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							resolver.CreateParameterName("builtin:management-zones", "342342-26re248-w46w48"): refParam.New("project", "builtin:management-zones", "342342-26re248-w46w48", "id"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", "something something -4292415658385853785 something something"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", "something something -4292415658385853785 something something"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", "something something mz-object-id something something"),
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
						Template:       template.NewInMemoryTemplate("4fw231-13fw124-f23r24", "content"),
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
						Template:       template.NewInMemoryTemplate("5242as-13fw124-f23r24", makeTemplateString("something something %s something something", "builtin:management-zones", "4fw231-13fw124-f23r24")),
						Coordinate:     coordinate.Coordinate{Project: "project", Type: "builtin:other-setting", ConfigId: "5242as-13fw124-f23r24"},
						OriginObjectId: "object2-objectID",
						Parameters: map[string]parameter.Parameter{
							config.ScopeParameter: valueParam.New("environment"),
							resolver.CreateParameterName("builtin:management-zones", "4fw231-13fw124-f23r24"): refParam.New("project", "builtin:management-zones", "4fw231-13fw124-f23r24", "id"),
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
						Template:   template.NewInMemoryTemplate("c1-id", `"template of config 1 references config 2: c2-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", `"template of config 2 references config 1: c1-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c1-id", makeTemplateString(`"template of config 1 references config 2: %s"`, "api", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api", "c2-id"): refParam.New("project", "api", "c2-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString(`"template of config 2 references config 1: %s"`, "api", "c1-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api", "c1-id"): refParam.New("project", "api", "c1-id", "id"),
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
						Template:   template.NewInMemoryTemplate("c1-id", `"template of config 1 references config 2: c2-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", `"template of config 2 references config 3: c3-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c3-id", `"template of config 3 references nothing"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c1-id", makeTemplateString(`"template of config 1 references config 2: %s"`, "api", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api", "c2-id"): refParam.New("project", "api", "c2-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString(`"template of config 2 references config 3: %s"`, "api", "c3-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api", "c3-id"): refParam.New("project", "api", "c3-id", "id"),
						},
					},
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c3-id", `"template of config 3 references nothing"`),
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
						Template:   template.NewInMemoryTemplate("c1-id", `"template of config 1 references config 2: c2-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{},
					},
				},
				"api-2": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-2"},
						Template:   template.NewInMemoryTemplate("c2-id", `"template of config 2 references config 3: c3-id"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "c2-id"},
						Parameters: config.Parameters{},
					},
				},
				"api-3": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-3"},
						Template:   template.NewInMemoryTemplate("c3-id", `"template of config 3 references nothing"`),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-3", ConfigId: "c3-id"},
						Parameters: config.Parameters{},
					},
				},
			},
			project.ConfigsPerType{
				"api": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api"},
						Template:   template.NewInMemoryTemplate("c1-id", makeTemplateString(`"template of config 1 references config 2: %s"`, "api-2", "c2-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "c1-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api-2", "c2-id"): refParam.New("project", "api-2", "c2-id", "id"),
						},
					},
				},
				"api-2": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-2"},
						Template:   template.NewInMemoryTemplate("c2-id", makeTemplateString(`"template of config 2 references config 3: %s"`, "api-3", "c3-id")),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "c2-id"},
						Parameters: config.Parameters{
							resolver.CreateParameterName("api-3", "c3-id"): refParam.New("project", "api-3", "c3-id", "id"),
						},
					},
				},
				"api-3": []config.Config{
					{
						Type:       config.ClassicApiType{Api: "api-3"},
						Template:   template.NewInMemoryTemplate("c3-id", `"template of config 3 references nothing"`),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api", "id2", "id"),
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "tenant"},
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id2"},
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "id3"},
						},
					},
				},
				"api-2": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("id3", ""),
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
						Template:   template.NewInMemoryTemplate("id1", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id1"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api", "id2", "id"),
						},
					},
					{
						Template:   template.NewInMemoryTemplate("id2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api", ConfigId: "id2"},
						Type:       config.SettingsType{SchemaId: "api"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "api-2", "id3", "id"),
						},
					},
				},
				"api-2": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("id3", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "api-2", ConfigId: "id3"},
						Type:       config.SettingsType{SchemaId: "api-2"},
						Parameters: config.Parameters{
							config.ScopeParameter: &valueParam.ValueParameter{Value: "environment"},
						},
					},
				},
			},
		},
		{
			name: "Dashboards should not be able to reference a dashboard-share-setting, even if it's the dashboard's share setting",
			setup: project.ConfigsPerType{
				"dashboard": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("t1", "dashboard-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id"},
						Type:       config.ClassicApiType{Api: "dashboard"},
						Parameters: config.Parameters{},
					},
					{
						Template:   template.NewInMemoryTemplate("t3", "dashboard-id"), // referencing the above dashboard
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id2"},
						Type:       config.ClassicApiType{Api: "dashboard"},
						Parameters: config.Parameters{},
					},
				},
				"dashboard-share-settings": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("t2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard-share-setting", ConfigId: "dashboard-id"},
						Type:       config.ClassicApiType{Api: "dashboard-share-setting"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "dashboard", "dashboard-id", "id"),
						},
					},
				},
			},
			expected: project.ConfigsPerType{
				"dashboard": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("t1", "dashboard-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id"},
						Type:       config.ClassicApiType{Api: "dashboard"},
						Parameters: config.Parameters{},
					},
					{
						Template:   template.NewInMemoryTemplate("t3", "dashboard-id"),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard", ConfigId: "dashboard-id2"},
						Type:       config.ClassicApiType{Api: "dashboard"},
						Parameters: config.Parameters{}, // no references to the other dashboard or dashboard-share-setting is allowed
					},
				},
				"dashboard-share-settings": []config.Config{
					{
						Template:   template.NewInMemoryTemplate("t2", ""),
						Coordinate: coordinate.Coordinate{Project: "project", Type: "dashboard-share-setting", ConfigId: "dashboard-id"},
						Type:       config.ClassicApiType{Api: "dashboard-share-setting"},
						Parameters: config.Parameters{
							config.ScopeParameter: refParam.New("project", "dashboard", "dashboard-id", "id"),
						},
					},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name+"_BasicResolver", func(t *testing.T) {
			result, err := ResolveDependencies(test.setup)
			require.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})

		t.Run(test.name+"_FastResolver", func(t *testing.T) {
			t.Setenv(featureflags.Permanent[featureflags.FastDependencyResolver].EnvName(), "true")
			result, err := ResolveDependencies(test.setup)
			require.NoError(t, err)
			assert.Equal(t, test.expected, result)
		})
	}
}

func makeTemplateString(template, api, configId string) string {
	return fmt.Sprintf(template, "{{."+resolver.CreateParameterName(api, configId)+"}}")
}
