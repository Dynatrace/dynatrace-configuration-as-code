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

// Copy copies everything from source into dest. Existing values are overwritten.
// dest may be nil if, and only if, source is empty or nil
func Copy[T comparable, V any, M ~map[T]V](dest, source M) {
	for k, v := range source {
		dest[k] = v
	}
}

// Keys returns all keys of the map
func Keys[K comparable, V any, M ~map[K]V](m M) []K {
	keys := make([]K, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

// Values returns all values of the map
func Values[K comparable, V any, M ~map[K]V](m M) []V {
	values := make([]V, 0, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}

// ToStringMap turns the Keys of a map[interface{}]interface{} into string keys
// will be transformed using fmt.Sprintf
func ToStringMap(m map[interface{}]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		result[fmt.Sprintf("%v", key)] = value
	}

	return result
}
