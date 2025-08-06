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
		input map[interface{}]interface{}
		want  map[string]interface{}
	}{
		{
			"parses empty map",
			map[interface{}]interface{}{},
			map[string]interface{}{},
		},
		{
			"parses string map",
			map[interface{}]interface{}{
				"one": "string",
				"two": "string",
			},
			map[string]interface{}{
				"one": "string",
				"two": "string",
			},
		},
		{
			"flattens non-strings",
			map[interface{}]interface{}{
				struct {
					Value  string
					Number int
				}{
					"something",
					42,
				}: 52,
			},
			map[string]interface{}{
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
			map[interface{}]interface{}{
				"property": map[interface{}]interface{}{
					"subproperty": "value",
				},
			},
			map[string]interface{}{
				"property": map[string]interface{}{
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
