// @license
// Copyright 2021 Dynatrace LLC
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

//go:build unit

package v2

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func Test_typeDefinition_isSound(t1 *testing.T) {
	type fields struct {
		configType typeDefinition
		knownApis  map[string]struct{}
	}
	type expect struct {
		result bool
		err    string
	}

	tests := []struct {
		name   string
		fields fields
		want   expect
	}{

		{
			"NO configuration at all",
			fields{
				*(new(typeDefinition)),
				nil,
			},
			expect{false, "type configuration is missing"},
		},
		{
			"Classical sound, settings incomplete",
			fields{
				typeDefinition{
					Api: "some.classical.api",
					Settings: settingsDefinition{
						Schema: "some.schema",
					},
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "wrong configuration of type property"},
		},
		{
			"Classical - sound",
			fields{
				typeDefinition{
					Api: "some.classical.api",
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{true, ""},
		},
		{
			"Classical - api is not known",
			fields{
				typeDefinition{
					Api: "not.known.api",
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "unknown API: not.known.api"},
		},
		{
			"Settings 2.0 - sound",
			fields{
				typeDefinition{
					Settings: settingsDefinition{
						Schema: "some.schema",
						Scope:  "scope",
					},
				},
				nil,
			},
			expect{true, ""},
		},
		{
			"Settings 2.0 - type.schema missing",
			fields{
				typeDefinition{
					Settings: settingsDefinition{
						Scope: "scope",
					},
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "property missing: [type.schema]"},
		},
		{
			"Settings 2.0 - type.scope missing",
			fields{
				typeDefinition{
					Settings: settingsDefinition{
						Schema: "some.schema",
					},
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "property missing: [type.scope]"},
		},
		{
			"Entity - sound",
			fields{
				typeDefinition{
					Entities: entitiesDefinition{
						EntitiesType: "SOMETHING",
					},
				},
				nil,
			},
			expect{true, ""},
		},
		{
			"Entity - EntitiesType missing",
			fields{
				typeDefinition{
					Entities: entitiesDefinition{},
				},
				nil,
			},
			expect{false, "type configuration is missing"},
		},
		{
			"Entity - wrong type",
			fields{
				typeDefinition{
					Api: "some.classical.api",
					Settings: settingsDefinition{
						Schema: "some.schema",
					},
					Entities: entitiesDefinition{
						EntitiesType: "SOMETHING",
					},
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "wrong configuration of type property"},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			configType := tt.fields.configType
			knownApis := tt.fields.knownApis

			actual, actualErr := configType.isSound(knownApis)
			assert.Equal(t1, actual, tt.want.result, tt.name)
			if tt.want.err != "" {
				assert.ErrorContains(t1, actualErr, tt.want.err, tt.name)
			}
		})
	}
}

func Test_typeDefinition_UnmarshalYAML(t *testing.T) {
	type given struct {
		ymlSample string
	}
	type expected struct {
		typeDefinition typeDefinition
		errorMessage   string
	}

	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name:  "shorthand syntax",
			given: given{"some.classical.api"},
			expected: expected{
				typeDefinition: typeDefinition{Api: "some.classical.api"},
			},
		},
		{
			name:  "Classical present",
			given: given{"Api: some.classical.api"},
			expected: expected{
				typeDefinition: typeDefinition{Api: "some.classical.api"},
			},
		},
		{
			name: "Settings 2.0 present",
			given: given{`
settings:
  schema: 'some.settings.schema'
  schemaVersion: '1.0'
  scope: 'scope'
`,
			},
			expected: expected{
				typeDefinition: typeDefinition{
					Settings: settingsDefinition{
						Schema:        "some.settings.schema",
						Scope:         "scope",
						SchemaVersion: "1.0",
					}},
			},
		},
		{
			name:  "wrong data type",
			given: given{"0x12d4"},
			expected: expected{
				errorMessage: "'type' section is not filed with proper values",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual typeDefinition
			err := yaml.Unmarshal([]byte(tt.given.ymlSample), &actual)

			if tt.expected.errorMessage == "" {
				assert.EqualValues(t, tt.expected.typeDefinition, actual)
			} else {
				assert.EqualError(t, err, tt.expected.errorMessage)
			}
		})
	}
}

func Test_typeDefinition_isSettings(t *testing.T) {
	tests := []struct {
		name  string
		given typeDefinition
		want  bool
	}{
		{
			name:  "empty struct",
			given: typeDefinition{},
			want:  false,
		},
		{
			name: "empty struct 2",
			given: typeDefinition{Settings: settingsDefinition{
				Schema:        "",
				SchemaVersion: "",
				Scope:         nil,
			}},
			want: false,
		},
		{
			name: "empty struct 2",
			given: typeDefinition{Settings: settingsDefinition{
				Schema:        "some.schema",
				SchemaVersion: "",
				Scope:         nil,
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.given.isSettings()
			assert.Equal(t, tt.want, actual)
		})
	}
}
