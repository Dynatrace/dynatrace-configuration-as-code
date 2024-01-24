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

package extract

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

// ConfigName extracts the value of the given config.Config's name parameter from the given parameter.Properties.
// If the name is not found in the properties, or the resolved value is not a string, ConfigName will return an error.
func ConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", errors.NewConfigDeployErr(conf, "missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", errors.NewConfigDeployErr(conf, "`name` in config is not of type string")
	}

	return name, nil
}

func Scope(properties parameter.Properties) (string, error) {
	scope, ok := properties[config.ScopeParameter]
	if !ok {
		return "", fmt.Errorf("property '%s' not found, this is most likely a bug", config.ScopeParameter)
	}

	switch v := scope.(type) {
	case string:
		if v == "" {
			return "", fmt.Errorf("resolved scope is empty")
		}
		return v, nil
	case []any:
		return "", fmt.Errorf("scope needs to be string, was a list")
	case map[any]any:
		return "", fmt.Errorf("scope needs to be string, was a map")
	default:
		return "", fmt.Errorf("scope needs to be string, was unexpected type %T", scope)
	}
}
