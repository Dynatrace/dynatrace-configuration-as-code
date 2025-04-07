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

package config

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

// resolveValues validates and resolves the given sorted parameters into actual values
func resolveValues(c *Config, entities EntityLookup, parameters []parameter.NamedParameter) (parameter.Properties, []error) {

	var errors []error

	properties := make(parameter.Properties)

	for _, container := range parameters {
		name := container.Name
		param := container.Parameter

		errs := validateParameterReferences(c.Coordinate, entities, name, param)

		if errs != nil {
			errors = append(errors, errs...)
			continue
		}

		val, err := param.ResolveValue(parameter.ResolveContext{
			PropertyResolver:        entities,
			ConfigCoordinate:        c.Coordinate,
			ParameterName:           name,
			ResolvedParameterValues: properties,
		})

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if name == NameParameter {
			properties[name] = strings.ToString(val)
		} else {
			properties[name] = val
		}
	}

	if len(errors) > 0 {
		// we want to return the partially resolved properties here, to find
		// more errors in the outer logic
		return properties, errors
	}

	return properties, nil
}

func validateParameterReferences(configCoordinates coordinate.Coordinate, entityLookup EntityLookup, paramName string, param parameter.Parameter) (errs []error) {

	for _, ref := range param.GetReferences() {
		// we have to ignore references to the same config,
		// as they will never be resolved before we validate
		// the parameters
		if ref.Config == configCoordinates {
			// parameters referencing themselves makes no sense
			if ref.Property == paramName {
				errs = append(errs, newParamsRefErr(configCoordinates, paramName, ref, "parameter referencing itself"))
			}

			continue
		}

		entity, found := entityLookup.GetResolvedEntity(ref.Config)

		if !found {
			errs = append(errs, newParamsRefErr(configCoordinates, paramName, ref, "referenced config not found"))
			continue
		}

		if entity.Skip {
			errs = append(errs, newParamsRefErr(configCoordinates, paramName, ref, "referencing skipped config"))
			continue
		}
	}

	return errs
}
