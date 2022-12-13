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
	"gotest.tools/assert"
	"testing"
)

func Test_configType_IsSound(t1 *testing.T) {
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
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			configType := tt.fields.configType
			knownApis := tt.fields.knownApis

			actual, actualErr := configType.IsSound(knownApis)
			assert.Equal(t1, actual, tt.want.result, tt.name)
			if tt.want.err != "" {
				assert.ErrorContains(t1, actualErr, tt.want.err, tt.name)
			}
		})
	}
}
