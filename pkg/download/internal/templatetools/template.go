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

package templatetools

import (
	"encoding/json"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

type JSONObject map[string]any

// NewJSONObject is a function that creates a JSONObject from a raw JSON byte slice.
func NewJSONObject(raw []byte) (JSONObject, error) {
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Get returns the value associated with a specified key in the JSONObject. If requested key doesn't exist, nil is returned.
func (o JSONObject) Get(key string) any {
	return o[key]
}

// Parameterize replaces the value associated with a specified key in the JSONObject with a template placeholder.
func (o JSONObject) Parameterize(key string) *value.ValueParameter {
	return o.ParameterizeAttributeWith(key, key)
}

// ParameterizeAttributeWith replace value of the given key with the given parameter name. The returned ValueParameter contains just replaced value for the given key.
func (o JSONObject) ParameterizeAttributeWith(keyOfJSONAttribute string, nameOfParameter string) *value.ValueParameter {
	if _, exits := o[keyOfJSONAttribute]; !exits {
		return nil
	}

	v := o[keyOfJSONAttribute]
	o[keyOfJSONAttribute] = "{{." + nameOfParameter + "}}"
	return &value.ValueParameter{Value: v}
}

// ToJSON converts JSONObject to its []byte representation.
// If pretty is true then the JSON is formatted with indentation and new lines
func (o JSONObject) ToJSON(pretty bool) ([]byte, error) {
	var bytes []byte
	var err error
	if pretty {
		bytes, err = json.MarshalIndent(o, "", "  ")
	} else {
		bytes, err = json.Marshal(o)
	}
	return bytes, err
}

// Delete removes a key-value pair for the specified key from JSONObject.
func (o JSONObject) Delete(keys ...string) {
	for _, k := range keys {
		delete(o, k)
	}
}
