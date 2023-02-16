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

// Filter return true iff specific api needs to be filtered/ removed from list
type Filter func(api Api) bool

// Filter apply all passed filters and return new filtered array
func (m ApiMap) Filter(filters ...Filter) ApiMap {
	apis := make(ApiMap)
	for k, v := range m {
		var keep = true
		for _, f := range filters {
			if f(v) && keep {
				keep = false
				break
			}
		}
		if keep {
			apis[k] = v
		}
	}
	return apis
}

// NoFilter is dummy filter that do nothing.
func NoFilter(Api) bool {
	return false
}

// RetainByName leave ony given apis. If api is not provided, nothing is removed.
func RetainByName(APIs []string) Filter {
	if len(APIs) == 0 {
		return NoFilter
	}

	return func(api Api) bool {
		for _, v := range APIs {
			if v == api.GetId() {
				return false
			}
		}
		return true
	}
}
