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

package filter

// FilterSlice filters elements in a slice based on a provided filter function.
// It iterates over each element in the input slice 'sl' and applies the 'filter' function to each element.
// If the 'filter' function returns true for an element, that element is included in the resulting slice.
// If the 'filter' function is nil, the original slice 'sl' is returned as is.
// The function returns a new slice containing the filtered elements.
func FilterSlice[T any](sl []T, filter func(T) bool) []T {
	result := make([]T, 0)
	if filter == nil {
		return sl
	}
	for _, item := range sl {
		if filter(item) {
			result = append(result, item)
		}
	}
	return result
}
