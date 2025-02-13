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

package configcreation

import (
	"encoding/json"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
)

type PreparedConfig struct {
	JSONString string
	Parameters map[string]parameter.Parameter
}

// PrepareConfig unmarshals the data into structPointer and modifies the JSON string based on deletedProperties and replaceParam
//
// The structPointer is basically forwarded to [json.Unmarshal]
//
// deletedProperties: Deletes properties of the given JSON data
//
// replaceParam: Replaces a property in the JSON with a placeholder
//
// Returns the modified JSON-string and a parameter map if replaceParam was specified
func PrepareConfig(data []byte, structPointer any, deletedProperties []string, replaceParam string) (PreparedConfig, error) {
	jsonObj, err := parseData(data, structPointer)

	if err != nil {
		return PreparedConfig{}, err
	}

	// delete fields that prevent a re-upload of the configuration
	for _, propertyKey := range deletedProperties {
		jsonObj.Delete(propertyKey)
	}

	parameters := parameterize(jsonObj, replaceParam)
	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return PreparedConfig{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return PreparedConfig{string(jsonRaw), parameters}, nil
}

// parseData unmarshals the data into structPointer and returns a new json object with the parsed data
func parseData(data []byte, structPointer any) (templatetools.JSONObject, error) {
	if err := json.Unmarshal(data, structPointer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	jsonObj, err := templatetools.NewJSONObject(data)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return jsonObj, nil
}

// parameterize replaces the value of a given property key with a placeholder and returns the original key-values as parameter map
func parameterize(jsonObj templatetools.JSONObject, replaceParam string) map[string]parameter.Parameter {
	parameters := make(map[string]parameter.Parameter)
	if replaceParam == "" {
		return nil
	}
	if p := jsonObj.Parameterize(replaceParam); p != nil {
		parameters[replaceParam] = p
	}
	return parameters
}
