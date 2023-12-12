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

package setting

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

type Validator struct{}

var deprecatedSchemas = map[string]string{
	"builtin:span-attribute":       "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
	"builtin:span-event-attribute": "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
	"builtin:resource-attribute":   "this setting was replaced by 'builtin:attribute-allow-list' and 'builtin:attribute-masking'",
}

// Validate checks for each settings type whether it is using a deprecated schema and if a value type scope is a non-empty string.
func (v *Validator) Validate(c config.Config) error {

	s, ok := c.Type.(config.SettingsType)
	if !ok {
		return nil
	}

	if msg, deprecated := deprecatedSchemas[s.SchemaId]; deprecated {
		log.WithFields(field.Coordinate(c.Coordinate), field.Environment(c.Environment, c.Group)).Warn("Schema %q is deprecated - please update your configurations: %s", s.SchemaId, msg)
	}

	return validateScopeType(c)
}

func validateScopeType(c config.Config) error {
	scope, ok := c.Parameters[config.ScopeParameter]
	if !ok {
		return fmt.Errorf("scope parameter not found")
	}

	if scope.GetType() != value.ValueParameterType {
		return nil // can't resolve other types without full information
	}

	val, err := scope.ResolveValue(parameter.ResolveContext{
		PropertyResolver: entities.New(),
		ConfigCoordinate: c.Coordinate,
		Group:            c.Group,
		Environment:      c.Environment,
		ParameterName:    config.ScopeParameter,
	})

	if err != nil {
		return fmt.Errorf("failed to resolve scope parameter: %w", err)
	}

	s, ok := val.(string)
	if !ok {
		return fmt.Errorf("scope needs to be a string: was unexpected type %T", scope)
	}

	if s == "" {
		return fmt.Errorf("resolved scope is empty")
	}

	return nil
}
