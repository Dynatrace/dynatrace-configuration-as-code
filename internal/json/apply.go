/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package json

import (
	"encoding/json"
)

// ApplyToStringValues unmarshals a JSON string and applies the specified transformation function to each string value before remarshaling and returning the result.
func ApplyToStringValues(jsonString string, f func(v string) string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(jsonString), &v); err != nil {
		return "", err
	}

	v = walkAnyAndApplyToStringValues(v, f)

	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func walkAnyAndApplyToStringValues(v any, f func(v string) string) any {
	switch vv := v.(type) {
	case string:
		if f == nil {
			return vv
		}
		return f(vv)

	case []interface{}:
		for i, u := range vv {
			vv[i] = walkAnyAndApplyToStringValues(u, f)
		}
		return vv

	case map[string]interface{}:
		for k, u := range vv {
			vv[k] = walkAnyAndApplyToStringValues(u, f)
		}
		return vv

	default:
		return vv
	}
}
