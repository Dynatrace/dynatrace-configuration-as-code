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

package api_test

import (
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/converter/v1environment"
	"github.com/stretchr/testify/assert"
)

var testDevEnvironment = v1environment.NewEnvironmentV1("development", "Dev", "", "https://url/to/dev/environment", "DEV")

func TestNewApis(t *testing.T) {
	apis := api.NewAPIs()

	assert.Contains(t, apis, "notification", "Expected `notification` key in KnownApis")
}

func TestContains(t *testing.T) {
	apis := api.NewAPIs()
	assert.True(t, apis.Contains("alerting-profile"))
	assert.False(t, apis.Contains("something"))

	assert.False(t, api.APIs{}.Contains("something"))
}

func TestApiMapFilter(t *testing.T) {
	type given struct {
		apis    api.APIs
		filters []api.Filter
	}
	type expected struct {
		apis api.APIs
	}
	tests := []struct {
		name     string
		given    given
		expected expected
	}{
		{
			name: "without filter",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: nil,
			},
			expected: expected{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
			},
		},
		{
			name: "filter with one filter",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{
					func(api api.API) bool { return api.ID == "api_1" },
				},
			},
			expected: expected{
				apis: api.APIs{
					"api_2": api.API{ID: "api_2"},
				},
			},
		},
		{
			name: "filter with two filters",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{
					api.Filter(func(api api.API) bool { return api.ID == "api_1" }),
					api.Filter(func(api api.API) bool { return api.ID == "api_2" }),
				},
			},
			expected: expected{
				apis: api.APIs{},
			},
		},
		{
			name: "noFilter",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{api.NoFilter},
			},
			expected: expected{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
			},
		},
		{
			name: "RetainByName - without arguments",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{api.RetainByName([]string{})},
			},
			expected: expected{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
			},
		},
		{
			name: "RetainByName - with arguments",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{api.RetainByName([]string{"api_1"})},
			},
			expected: expected{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
				},
			},
		}, {
			name: "RetainByName - with non existing argument",
			given: given{
				apis: api.APIs{
					"api_1": api.API{ID: "api_1"},
					"api_2": api.API{ID: "api_2"},
				},
				filters: []api.Filter{api.RetainByName([]string{"api_3"})},
			},
			expected: expected{
				apis: api.APIs{},
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
