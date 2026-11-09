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

	"github.com/google/go-cmp/cmp"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	compoundParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/compound"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
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

// Validate checks that for each classic config API type and each type in [config.ConfigsIdentifiedByName], only one config exists with any given name.
// As these configs are identified by name, Validate returns errors if a name is used more than once for the same type.
func (v *validator) Validate(_ []project.Project, c config.Config) error {
	if v.uniqueNames == nil {
		v.uniqueNames = make(map[environmentName]map[classicEndpoint][]config.Config)
	}

	key, ok := v.nameIdentifiedTypeKey(c)
	if !ok {
		return nil
	}

	if v.uniqueNames[c.Environment] == nil {
		v.uniqueNames[c.Environment] = make(map[classicEndpoint][]config.Config)
	}

	for _, c2 := range v.uniqueNames[c.Environment][key] {
		if !scopesAreEqual(&c, &c2) {
			continue
		}

		if err := ensureNonDuplicateNames(c, c2); err != nil {
			return err
		}
	}

	v.uniqueNames[c.Environment][key] = append(v.uniqueNames[c.Environment][key], c)
	return nil
}

// nameIdentifiedTypeKey returns the key under which names of c must be unique, or false if c doesn't require unique names.
func (v *validator) nameIdentifiedTypeKey(c config.Config) (string, bool) {
	if c.Type == nil {
		return "", false
	}

	a, ok := c.Type.(config.ClassicApiType)
	if !ok {
		typeID := string(c.Type.ID())
		return typeID, config.ConfigsIdentifiedByName[typeID]
	}

	if v.apis[a.Api].NonUniqueName {
		return "", false
	}

	// as the uniqueness of a key-user-action-web configuration is defined by its payload no validation can be performed
	if a.Api == api.KeyUserActionsWeb {
		return "", false
	}

	return a.Api, true
}

func scopesAreEqual(c1 *config.Config, c2 *config.Config) bool {
	// if the configs have a scope, and they are different, then the configs don't need to be checked further for name-uniqueness
	scope1 := c1.Parameters[config.ScopeParameter]
	scope2 := c2.Parameters[config.ScopeParameter]

	return cmp.Equal(scope1, scope2)
}

// ensureNonDuplicateNames returns an error if a name is used more than once for the same type.
// If parameter values can be resolved for both configs, we compare name parameters directly.
// If they can't be resolved, we compare the unresolved NameParameters of the two configs,
// first assuming both being of type ReferenceParameter, and if that fails, assuming both being of type CompoundParameter.
// If the fields are equal, a name clash is guaranteed.
// This check is not perfect. Name clashes are not caught if, e.g., one config is resolvable and the other one isn't.
func ensureNonDuplicateNames(c config.Config, c2 config.Config) error {
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
	return nil
}

// deprecatedApiValidator should be created via NewDeprecatedApiValidator().
// For a given classic API config, this validator checks the specific API.
// If this API is deprecated, and the validator has not seen that API before, it will log a warning.
type deprecatedApiValidator struct {
	apis     api.APIs
	seenApis map[string]struct{}
}

func NewDeprecatedApiValidator() *deprecatedApiValidator {
	return &deprecatedApiValidator{
		apis:     api.NewAPIs(),
		seenApis: make(map[string]struct{}),
	}
}

// Validate checks if the given config API is deprecated. If it is, and if the validator has not seen the api before, a warning will be logged.
func (v *deprecatedApiValidator) Validate(_ []project.Project, c config.Config) error {

	a, ok := c.Type.(config.ClassicApiType)
	if !ok {
		return nil
	}

	if _, apiAlreadySeen := v.seenApis[a.Api]; apiAlreadySeen {
		return nil
	}

	v.seenApis[a.Api] = struct{}{}
	theAPI := v.apis[a.Api]

	if theAPI.DeprecatedBy != "" {
		log.Warn("API '%s' is deprecated. Please migrate to '%s'.", theAPI.ID, theAPI.DeprecatedBy)
	}

	return nil
}
