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
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
)

// PrepareConfig returns the given id (idPropertyKey) and the JSON string without the given properties
func PrepareConfig(data []byte, idPropertyKey string, deletedProperties ...string) (string, string, error) {
	jsonObj, err := templatetools.NewJSONObject(data)
	if err != nil {
		return "", "", fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get(idPropertyKey).(string)
	if !ok {
		return "", "", fmt.Errorf("API payload is missing '%s'", idPropertyKey)
	}

	// delete fields that prevent a re-upload of the configuration
	for _, propertyKey := range deletedProperties {
		jsonObj.Delete(propertyKey)
	}

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	return id, string(jsonRaw), nil
}
