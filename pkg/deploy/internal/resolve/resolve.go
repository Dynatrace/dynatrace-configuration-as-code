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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/resolve"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/entitymap"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
)

func Properties(c *config.Config, entityMap *entitymap.EntityMap) (parameter.Properties, []error) {
	var errors []error

	parameters, sortErrs := sort.Parameters(c.Group, c.Environment, c.Coordinate, c.Parameters)
	errors = append(errors, sortErrs...)

	properties, errs := resolve.ParameterValues(c, entityMap, parameters)
	errors = append(errors, errs...)

	if len(errors) > 0 {
		return nil, errors
	}

	return properties, nil
}

func ExtractConfigName(conf *config.Config, properties parameter.Properties) (string, error) {
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
