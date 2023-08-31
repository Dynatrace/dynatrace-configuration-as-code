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

package persistence

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"testing"
)

func Test_typeDefinition_isSound(t *testing.T) {
	t.Setenv(featureflags.Buckets().EnvName(), "1")

	type fields struct {
		configType TypeDefinition
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
				*(new(TypeDefinition)),
				nil,
			},
			expect{false, "type configuration is missing"},
		},
		{
			"Classical sound, settings incomplete",
			fields{
				TypeDefinition{
					Api: "some.classical.api",
					Settings: SettingsDefinition{
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
				TypeDefinition{
					Api: "some.classical.api",
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{true, ""},
		},
		{
			"Classical - api is not known",
			fields{
				TypeDefinition{
					Api: "not.known.api",
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "unknown API: not.known.api"},
		},
		{
			"Settings 2.0 - sound",
			fields{
				TypeDefinition{
					Settings: SettingsDefinition{
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
				TypeDefinition{
					Settings: SettingsDefinition{
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
				TypeDefinition{
					Settings: SettingsDefinition{
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
				TypeDefinition{
					Entities: EntitiesDefinition{
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
				TypeDefinition{
					Entities: EntitiesDefinition{},
				},
				nil,
			},
			expect{false, "type configuration is missing"},
		},
		{
			"Entity - wrong type",
			fields{
				TypeDefinition{
					Api: "some.classical.api",
					Settings: SettingsDefinition{
						Schema: "some.schema",
					},
					Entities: EntitiesDefinition{
						EntitiesType: "SOMETHING",
					},
				},
				map[string]struct{}{"some.classical.api": {}},
			},
			expect{false, "wrong configuration of type property"},
		},
		{
			name: "Bucket - sound",
			fields: fields{
				configType: TypeDefinition{
					Bucket: "bucket",
				},
				knownApis: map[string]struct{}{},
			},
			want: expect{
				result: true,
			},
		},
		{
			name: "Bucket - as API is invalid",
			fields: fields{
				configType: TypeDefinition{
					Api: "bucket",
				},
				knownApis: map[string]struct{}{},
			},
			want: expect{
				result: false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configType := tt.fields.configType
			knownApis := tt.fields.knownApis

			actualErr := configType.IsSound(knownApis)
			assert.Equal(t, actualErr == nil, tt.want.result, tt.name)
			if tt.want.err != "" {
				assert.ErrorContains(t, actualErr, tt.want.err, tt.name)
			}
		})
	}
}

func Test_typeDefinition_UnmarshalYAML(t *testing.T) {
	type given struct {
		ymlSample string
	}
	type expected struct {
		typeDefinition TypeDefinition
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
				typeDefinition: TypeDefinition{Api: "some.classical.api"},
			},
		},
		{
			name:  "Classical present",
			given: given{"Api: some.classical.api"},
			expected: expected{
				typeDefinition: TypeDefinition{Api: "some.classical.api"},
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
				typeDefinition: TypeDefinition{
					Settings: SettingsDefinition{
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
				errorMessage: "failed to parse 'type' section: '' expected a map, got 'int'",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual TypeDefinition
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
		given TypeDefinition
		want  bool
	}{
		{
			name:  "empty struct",
			given: TypeDefinition{},
			want:  false,
		},
		{
			name: "empty struct 2",
			given: TypeDefinition{Settings: SettingsDefinition{
				Schema:        "",
				SchemaVersion: "",
				Scope:         nil,
			}},
			want: false,
		},
		{
			name: "empty struct 2",
			given: TypeDefinition{Settings: SettingsDefinition{
				Schema:        "some.schema",
				SchemaVersion: "",
				Scope:         nil,
			}},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.given.IsSettings()
			assert.Equal(t, tt.want, actual)
		})
	}
}
