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

package loader

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account/internal/types"
)

// validateReferences checks the references in the provided AMResources instance to ensure
// that all referenced groups and policies exist. It iterates through the users,
// environment policies, and account policies, validating their references.
func validateReferences(res *types.Resources) error {
	for _, user := range res.Users {
		for _, groupRef := range user.Groups {
			if err := refCheck(res, groupRef, groupExists); err != nil {
				return err
			}
		}
	}

	for _, group := range res.Groups {
		// check references in environment policies
		for _, env := range group.Environment {
			for _, policyRef := range env.Policies {
				if err := refCheck(res, policyRef, policyExists); err != nil {
					return err
				}
			}
		}
		if group.Account != nil {
			// check references in account policies
			for _, policyRef := range group.Account.Policies {
				if err := refCheck(res, policyRef, policyExists); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func refCheck(res *types.Resources, reference types.Reference, refCheckFn func(*types.Resources, string) bool) error {
	if reference.Type == types.ReferenceType {
		if reference.Id == "" {
			return fmt.Errorf("error validating account resources: %w", ErrIdFieldMissing)
		}

		refExists := refCheckFn(res, reference.Id)
		if !refExists {
			return fmt.Errorf("error validating account resources with id %q: %w", reference.Id, ErrRefMissing)
		}
	} else if reference.Value == "" {
		return fmt.Errorf("error validating account resources: %w", errors.New("value is missing"))
	}
	return nil
}

func groupExists(a *types.Resources, id string) bool {
	_, exists := a.Groups[id]
	return exists
}

func policyExists(a *types.Resources, id string) bool {
	_, exists := a.Policies[id]
	return exists

}

func validateUser(u types.User) error {
	if u.Email == "" {
		return errors.New("missing required field 'email' for user")
	}
	return nil
}

func validateGroup(g types.Group) error {
	if g.ID == "" {
		return errors.New("missing required field 'id' for policy")
	}
	if g.Name == "" {
		return fmt.Errorf("missing required field 'name' for group %q", g.ID)
	}
	// groups don't necessarily need to define any permissions/policies
	return nil
}

func validatePolicy(p types.Policy) error {
	if p.ID == "" {
		return errors.New("missing required field 'id' for policy")
	}
	if p.Name == "" {
		return fmt.Errorf("missing required field 'name' for policy %q", p.ID)
	}
	if p.Level.Type == "" {
		return fmt.Errorf("missing required field 'level.type' for policy %q", p.ID)
	}
	if p.Level.Type != types.PolicyLevelAccount && p.Level.Type != types.PolicyLevelEnvironment {
		return fmt.Errorf("unknown 'level.type' for policy %q: %q (needs to be one of: %q, %q)", p.ID, p.Level.Type, types.PolicyLevelAccount, types.PolicyLevelEnvironment)
	}
	if p.Level.Type == types.PolicyLevelEnvironment && p.Level.Environment == "" {
		return fmt.Errorf("missing required field 'level.environment' for environment-level policy %q", p.ID)
	}
	if p.Policy == "" {
		return fmt.Errorf("missing required field 'policy' for policy %q", p.ID)
	}
	return nil
}
