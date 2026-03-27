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

package maps

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToStringMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input map[any]any
		want  map[string]any
	}{
		{
			"parses empty map",
			map[any]any{},
			map[string]any{},
		},
		{
			"parses string map",
			map[any]any{
				"one": "string",
				"two": "string",
			},
			map[string]any{
				"one": "string",
				"two": "string",
			},
		},
		{
			"flattens non-strings",
			map[any]any{
				struct {
					Value  string
					Number int
				}{
					"something",
					42,
				}: 52,
			},
			map[string]any{
				fmt.Sprintf("%v", struct {
					Value  string
					Number int
				}{
					"something",
					42,
				}): 52,
			},
		},
		{
			"Applies the stringify recursively",
			map[any]any{
				"property": map[any]any{
					"subproperty": "value",
				},
			},
			map[string]any{
				"property": map[string]any{
					"subproperty": "value",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToStringMap(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}
