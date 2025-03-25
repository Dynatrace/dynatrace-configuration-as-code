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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	compoundParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"

	"github.com/google/go-cmp/cmp"
)

type (
	environmentName = string
	classicEndpoint = string
)

// validator should be created via NewValidator().
type validator struct {
	apis        api.APIs
	uniqueNames map[environmentName]map[classicEndpoint][]config.Config
}

// NewValidator creates a new validator.
func NewValidator() *validator {
	return &validator{
		apis: api.NewAPIs(),
	}
}

// Validate checks that for each classic config API type, only one config exists with any given name.
// As classic configs are identified by name, ValidateUniqueConfigNames returns errors if a name is used more than once for the same type.
func (v *validator) Validate(_ []project.Project, c config.Config) error {
	if v.uniqueNames == nil {
		v.uniqueNames = make(map[environmentName]map[classicEndpoint][]config.Config)
	}

	a, ok := c.Type.(config.ClassicApiType)
	if !ok {
		return nil
	}

	theAPI := v.apis[a.Api]
	if theAPI.NonUniqueName {
		return nil
	}

	// as the uniqueness of a key-user-action-web configuration is defined by its payload no validation can be performed
	if a.Api == api.KeyUserActionsWeb {
		return nil
	}

	if v.uniqueNames[c.Environment] == nil {
		v.uniqueNames[c.Environment] = make(map[classicEndpoint][]config.Config)
	}

	for _, c2 := range v.uniqueNames[c.Environment][a.Api] {
		// if the configs have a scope and they are different then the configs are unique
		scope1 := c.Parameters[config.ScopeParameter]
		scope2 := c2.Parameters[config.ScopeParameter]
		if !cmp.Equal(scope1, scope2) {
			continue
		}

		// if we are able to resolve the parameters, compare their value
		cResolvedParams, errc := c.ResolveParameterValues(entities.New())
		c2ResolvedParams, errc2 := c2.ResolveParameterValues(entities.New())
		if len(errc) == 0 && len(errc2) == 0 {
			if cmp.Equal(cResolvedParams[config.NameParameter], c2ResolvedParams[config.NameParameter]) {
				return fmt.Errorf("duplicated config name found: configurations %s and %s define the same 'name' %q", c.Coordinate, c2.Coordinate, cResolvedParams[config.NameParameter])
			}
		}

		// check if (unresolvable) reference parameters are equal
		if r, ok := c.Parameters[config.NameParameter].(*reference.ReferenceParameter); ok {
			if r2, ok := c2.Parameters[config.NameParameter].(*reference.ReferenceParameter); ok {
				if r.Equal(r2) {
					return fmt.Errorf("duplicated config name found: configurations %s and %s define the same 'name'", c.Coordinate, c2.Coordinate)
				}
			}
		}

		// check if (unresolvable) compound parameters are equal
		if r, ok := c.Parameters[config.NameParameter].(*compoundParam.CompoundParameter); ok {
			if r2, ok := c2.Parameters[config.NameParameter].(*compoundParam.CompoundParameter); ok {
				if r.Equal(r2) {
					return fmt.Errorf("duplicated config name found: configurations %s and %s define the same 'name'", c.Coordinate, c2.Coordinate)
				}
			}
		}
	}

	v.uniqueNames[c.Environment][a.Api] = append(v.uniqueNames[c.Environment][a.Api], c)
	return nil
}
