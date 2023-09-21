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

package raw

import (
	"encoding/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

type JSONObject map[string]any

func New(raw []byte) (JSONObject, error) {
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (o JSONObject) Get(key string) any {
	return o[key]
}

func (o JSONObject) ParameterizeAttribute(keyOfJSONAttribute string, nameOfParameter string) *value.ValueParameter {
	if _, exits := o[keyOfJSONAttribute]; !exits {
		return nil
	}

	v := o[keyOfJSONAttribute]
	o[keyOfJSONAttribute] = "{{." + nameOfParameter + "}}"
	return &value.ValueParameter{Value: v}
}

func (o JSONObject) ToJSON() ([]byte, error) {
	modified, err := json.Marshal(o)
	if err != nil {
		return []byte{}, err
	}

	return modified, nil
}
