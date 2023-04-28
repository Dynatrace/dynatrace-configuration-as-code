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

// ToStringMap turns the Keys of a map[interface{}]interface{} into string keys
// will be transformed using fmt.Sprintf
func ToStringMap(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		result[fmt.Sprintf("%v", key)] = value
	}

	return result
}
