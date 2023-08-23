// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
)

type EntityLookup interface {
	parameter.PropertyResolver

	Entity(config coordinate.Coordinate) (ResolvedEntity, bool)
}

// TODO: unexport this function
func ResolveParameterValues(
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

func resolveProperties(c *config.Config, entities *entityMap) (parameter.Properties, []error) {
	var errors []error

	parameters, sortErrs := sort.Parameters(c.Group, c.Environment, c.Coordinate, c.Parameters)
	errors = append(errors, sortErrs...)

	properties, errs := ResolveParameterValues(c, entities, parameters)
	errors = append(errors, errs...)

	if len(errors) > 0 {
		return nil, errors
	}

	return properties, nil
}

func validateParameterReferences(configCoordinates coordinate.Coordinate,
	group string, environment string,
	entities EntityLookup,
	paramName string,
	param parameter.Parameter,
) (errors []error) {

	for _, ref := range param.GetReferences() {
		// we have to ignore references to the same config,
		// as they will never be resolved before we validate
		// the parameters
		if ref.Config == configCoordinates {
			// parameters referencing themselves makes no sense
			if ref.Property == paramName {
				errors = append(errors, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "parameter referencing itself"))
			}

			continue
		}

		entity, found := entities.Entity(ref.Config)

		if !found {
			errors = append(errors, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "referenced config not found"))
			continue
		}

		if entity.Skip {
			errors = append(errors, newParamsRefErr(configCoordinates, group, environment, paramName, ref, "referencing skipped config"))
			continue
		}
	}

	return errors
}

func extractConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", newConfigDeployErr(conf, "missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", newConfigDeployErr(conf, "`name` in config is not of type string")
	}

	return name, nil
}
