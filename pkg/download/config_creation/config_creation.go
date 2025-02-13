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

package config_creation

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

// PrepareConfig returns the given id (idPropertyKey) and the JSON string without the given properties
func PrepareConfig(data []byte, structPointer any, deletedProperties []string, replaceParam string) (PreparedConfig, error) {
	if err := json.Unmarshal(data, structPointer); err != nil {
		return PreparedConfig{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	jsonObj, err := templatetools.NewJSONObject(data)
	parameters := map[string]parameter.Parameter{}

	if err != nil {
		return PreparedConfig{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// delete fields that prevent a re-upload of the configuration
	for _, propertyKey := range deletedProperties {
		jsonObj.Delete(propertyKey)
	}

	if replaceParam != "" {
		if p := jsonObj.Parameterize(replaceParam); p != nil {
			parameters[replaceParam] = p
		}
	}

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return PreparedConfig{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return PreparedConfig{string(jsonRaw), parameters}, nil
}
