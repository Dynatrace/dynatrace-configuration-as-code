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

package resolve

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
)

// EntityLookup is used in parameter resolution to fetch the resolved entity of deployed configuration
type EntityLookup interface {
	parameter.PropertyResolver

	GetResolvedEntity(config coordinate.Coordinate) (config.ResolvedEntity, bool)
}

// ParameterValues will resolve the values of all config.Parameters of a config.Config c and return them as a parameter.Properties map.
// Resolving will ensure that parameters are resolved in the right order if they have dependencies between each other.
// To be able to resolve reference.ReferenceParameter values an EntityLookup needs to be provided, which contains all
// config.ResolvedEntity values of configurations that config.Config c could depend on.
// Ordering of configurations to ensure that possible dependency configurations are contained in teh EntityLookup is responsibility
// of the caller of ParameterValues.
//
// ParameterValues will return a slice of errors for any failures during sorting or resolving parameters.
func ParameterValues(c *config.Config, entities EntityLookup) (parameter.Properties, []error) {
	var errors []error

	parameters, sortErrs := sort.Parameters(c.Group, c.Environment, c.Coordinate, c.Parameters)
	errors = append(errors, sortErrs...)

	properties, errs := resolveValues(c, entities, parameters)
	errors = append(errors, errs...)

	if len(errors) > 0 {
		return nil, errors
	}

	return properties, nil
}

func resolveValues(
	conf *config.Config,
	entities EntityLookup,
	parameters []parameter.NamedParameter,
) (parameter.Properties, []error) {

	var errors []error

	properties := make(parameter.Properties)

	for _, container := range parameters {
		name := container.Name
		param := container.Parameter

		errs := validateParameterReferences(conf.Coordinate, conf.Group, conf.Environment, entities, name, param)

		if errs != nil {
			errors = append(errors, errs...)
			continue
		}

		val, err := param.ResolveValue(parameter.ResolveContext{
			PropertyResolver:        entities,
			ConfigCoordinate:        conf.Coordinate,
			Group:                   conf.Group,
			Environment:             conf.Environment,
			ParameterName:           name,
			ResolvedParameterValues: properties,
		})

		if err != nil {
			errors = append(errors, err)
			continue
		}

		if name == config.NameParameter {
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

func validateParameterReferences(configCoordinates coordinate.Coordinate,
	group string, environment string,
	entityLookup EntityLookup,
	paramName string,
	param parameter.Parameter,
) (errs []error) {

	for _, ref := range param.GetReferences() {
		// we have to ignore references to the same config,
		// as they will never be resolved before we validate
		// the parameters
		if ref.Config == configCoordinates {
			// parameters referencing themselves makes no sense
			if ref.Property == paramName {
				errs = append(errs, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "parameter referencing itself"))
			}

			continue
		}

		entity, found := entityLookup.GetResolvedEntity(ref.Config)

		if !found {
			errs = append(errs, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "referenced config not found"))
			continue
		}

		if entity.Skip {
			errs = append(errs, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "referencing skipped config"))
			continue
		}
	}

	return errs
}
