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

package slices

import "slices"

// Difference is removing all elements from a which are present in b.
// It is the mathematical set-operation equivalent as (A - B)
func Difference[T comparable, S ~[]T](a S, b S) S {
	result := make(S, 0, len(a))

	// Could be optimized for larger slices by using maps, or sorting & iterating. Not needed for now
	for _, v := range a {
		if !slices.Contains(b, v) {
			result = append(result, v)
		}
	}

	return result
}
