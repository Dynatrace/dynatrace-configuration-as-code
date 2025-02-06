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
// If no transformation actually occurs, the original JSON string is returned.
func ApplyToStringValues(jsonString string, applyFunc func(v string) string) (string, error) {
	var val any
	if err := json.Unmarshal([]byte(jsonString), &val); err != nil {
		return "", err
	}

	val, changed := walkAnyAndApplyToStringValues(val, applyFunc)
	if !changed {
		return jsonString, nil
	}

	newJson, err := json.Marshal(val)
	if err != nil {
		return "", err
	}
	return string(newJson), nil
}

// walkAnyAndApplyToStringValues recursively visits each node and applies the specified transformation to each string value.
// The updated value is returned as a well as a boolean indicating if the value was changed.
func walkAnyAndApplyToStringValues(v any, applyFunc func(v string) string) (any, bool) {
	switch typ := v.(type) {
	case string:
		if applyFunc == nil {
			return typ, false
		}
		fNew := applyFunc(typ)
		return fNew, typ != fNew

	case []any:
		changed := false
		for i, u := range typ {
			uNew, c := walkAnyAndApplyToStringValues(u, applyFunc)
			typ[i] = uNew
			changed = changed || c
		}
		return typ, changed

	case map[string]any:
		changed := false
		for k, u := range typ {
			uNew, c := walkAnyAndApplyToStringValues(u, applyFunc)
			typ[k] = uNew
			changed = changed || c
		}
		return typ, changed

	default:
		return typ, false
	}
}
