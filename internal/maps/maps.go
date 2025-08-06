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

import "fmt"

// ToStringMap turns the Keys of a map[any]any into string keys.
// Keys will be transformed using fmt.Sprint.
// This function works recursively, so nested maps will be converted as well.
func ToStringMap(original map[any]any) map[string]any {
	result := make(map[string]any)

	for key, value := range original {
		// recursively convert all 'map[any]any' to 'map[string]any'
		if subMap, ok := value.(map[any]any); ok {
			value = ToStringMap(subMap)
		}

		result[fmt.Sprint(key)] = value
	}

	return result
}
