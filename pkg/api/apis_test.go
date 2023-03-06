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

package api

import "testing"
import "github.com/stretchr/testify/assert"

func TestContains(t *testing.T) {
	apis := NewApis()
	assert.True(t, apis.Contains("alerting-profile"))
	assert.False(t, apis.Contains("something"))

	assert.False(t, APIs{}.Contains("something"))
}

func TestContainsApiName(t *testing.T) {
	apis := NewApis()
	assert.False(t, apis.ContainsApiName("trillian"), "Check if `trillian` is an API")
	assert.True(t, apis.ContainsApiName("extension"), "Check if `extension` is an API")
	assert.True(t, apis.ContainsApiName("/project/sub-project/extension/subfolder"), "Check if `extension` is an API")
	assert.False(t, apis.ContainsApiName("/project/sub-project"), "Check if `extension` is an API")
}

func TestApiMapFilter(t *testing.T) {
	type given struct {
		apis    APIs
		filters []Filter
	}
	type expected struct {
		apis APIs
	}
	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name: "without filter",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: nil,
			},
			expected: expected{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
			},
		},
		{
			name: "filter with one filter",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{
					func(api *Api) bool { return api.GetId() == "api_1" },
				},
			},
			expected: expected{
				apis: APIs{
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
			},
		},
		{
			name: "filter with two filters",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{
					Filter(func(api *Api) bool { return api.GetId() == "api_1" }),
					Filter(func(api *Api) bool { return api.GetId() == "api_2" }),
				},
			},
			expected: expected{
				apis: APIs{},
			},
		},
		{
			name: "NoFilter",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{NoFilter},
			},
			expected: expected{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
			},
		},
		{
			name: "RetainByName - without arguments",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{RetainByName([]string{})},
			},
			expected: expected{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
			},
		},
		{
			name: "RetainByName - with arguments",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{RetainByName([]string{"api_1"})},
			},
			expected: expected{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
				},
			},
		}, {
			name: "RetainByName - with non existing argument",
			given: given{
				apis: APIs{
					"api_1": NewApi("api_1", "", "", false, false, "", false),
					"api_2": NewApi("api_2", "", "", false, false, "", false),
				},
				filters: []Filter{RetainByName([]string{"api_3"})},
			},
			expected: expected{
				apis: APIs{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.given.apis.Filter(tt.given.filters...)
			assert.Equal(t, tt.expected.apis, actual)
		})
	}
}
