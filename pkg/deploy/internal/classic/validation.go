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

package classic

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/google/go-cmp/cmp"
)

// ValidateUniqueConfigNames checks that for each classic config API type, only one config exists with any given name.
// As classic configs are identified by name, ValidateUniqueConfigNames returns errors if a name is used more than once for the same type.
func ValidateUniqueConfigNames(projects []project.Project) error {
	errs := make(errors.EnvironmentDeploymentErrors)
	type (
		environmentName = string
		classicEndpoint = string
	)
	uniqueList := make(map[environmentName]map[classicEndpoint][]config.Config)
	e := api.NewAPIs()

	checkUniquenessOfName := func(c config.Config) {
		a, ok := c.Type.(config.ClassicApiType)
		if !ok || e[a.Api].NonUniqueName {
			return
		}

		if uniqueList[c.Environment] == nil {
			uniqueList[c.Environment] = make(map[classicEndpoint][]config.Config)
		}

		for _, c2 := range uniqueList[c.Environment][a.Api] {
			n1, err := getNameForConfig(c)
			if err != nil {
				errs = errs.Append(c.Environment, err)
				return
			}
			n2, err := getNameForConfig(c2)
			if err != nil {
				errs = errs.Append(c.Environment, err)
				return
			}

			if cmp.Equal(n1, n2) {
				errs = errs.Append(c.Environment, fmt.Errorf("configuration with coordinates %q and %q have same \"name\" values", c.Coordinate, c2.Coordinate))
				return
			}
		}

		uniqueList[c.Environment][a.Api] = append(uniqueList[c.Environment][a.Api], c)
	}

	for _, p := range projects {
		p.ForEveryConfigDo(checkUniquenessOfName)
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

func getNameForConfig(c config.Config) (any, error) {
	nameParam, exist := c.Parameters[config.NameParameter]
	if !exist {
		return nil, fmt.Errorf("config %s has no 'name' parameter defined", c.Coordinate)
	}

	switch v := nameParam.(type) {
	case *value.ValueParameter:
		return v.ResolveValue(parameter.ResolveContext{ParameterName: config.NameParameter})
	case *environment.EnvironmentVariableParameter:
		return v.ResolveValue(parameter.ResolveContext{ParameterName: config.NameParameter})
	default:
		return c.Parameters[config.NameParameter], nil
	}
}
