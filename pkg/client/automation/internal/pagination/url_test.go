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

package pagination_test

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation/internal/pagination"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_loadMore(t *testing.T) {

	type given struct {
		baseURL string
		path    string
		offset  int
	}

	tests := []struct {
		name     string
		given    given
		expected string
	}{
		{
			name:     "simple case",
			given:    given{baseURL: "https://base.url", path: "", offset: 0},
			expected: "https://base.url?offset=0",
		},
		{
			name:     "url have query params",
			given:    given{baseURL: "https://base.url?param=exits", path: "", offset: 0},
			expected: "https://base.url?offset=0&param=exits",
		},
		{
			name:     "add path",
			given:    given{baseURL: "https://base.url/a/b/", path: "new/path", offset: 0},
			expected: "https://base.url/a/b/new/path?offset=0",
		},
		{
			name:     "add path - baseURL without end-slash",
			given:    given{baseURL: "https://base.url/a", path: "new/path", offset: 0},
			expected: "https://base.url/a/new/path?offset=0",
		},
		{
			name:     "add path - path with starting-slash",
			given:    given{baseURL: "https://base.url/a/", path: "/new/path", offset: 0},
			expected: "https://base.url/a/new/path?offset=0",
		},
		{
			name:     "modified offset",
			given:    given{baseURL: "https://base.url", path: "", offset: 42},
			expected: "https://base.url?offset=42",
		},
		{
			name:     "full case",
			given:    given{baseURL: "https://base.url?param=exits", path: "new/path", offset: 42},
			expected: "https://base.url/new/path?offset=42&param=exits",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := pagination.NextPageURL(tc.given.baseURL, tc.given.path, tc.given.offset)

			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
