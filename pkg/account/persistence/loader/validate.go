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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/internal/types"
)

// validateReferences checks the references in the provided AMResources instance to ensure
// that all referenced groups and policies exist. It iterates through the users,
// environment policies, and account policies, validating their references.
func validateReferences(res *account.Resources) error {
	for _, user := range res.Users {
		if err := validateUserReferences(res, user); err != nil {
			return err
		}
	}

	for _, serviceUser := range res.ServiceUsers {
		if err := validateServiceUserReferences(res, serviceUser); err != nil {
			return err
		}
	}

	for _, group := range res.Groups {
		if err := validateGroupReferences(res, group); err != nil {
			return err
		}
	}
	return nil
}

func validateUserReferences(res *account.Resources, user account.User) error {
	for _, groupRef := range user.Groups {
		groupReference, ok := groupRef.(account.Reference)
		if !ok {
			continue
		}

		if !groupExists(res, groupReference.ID()) {
			return fmt.Errorf("user '%s' references missing group '%s'", user.Email, groupReference.ID())
		}
	}
	return nil
}

func validateServiceUserReferences(res *account.Resources, serviceUser account.ServiceUser) error {
	for _, groupRef := range serviceUser.Groups {
		groupReference, ok := groupRef.(account.Reference)
		if !ok {
			continue
		}

		if !groupExists(res, groupReference.ID()) {
			return fmt.Errorf("service user '%s' references missing group '%s'", serviceUser.Name, groupReference.ID())
		}
	}
	return nil
}

func validateGroupReferences(res *account.Resources, group account.Group) error {
	// check references in environment policies
	for _, env := range group.Environment {
		for _, policyRef := range env.Policies {
			policyReference, ok := policyRef.(account.Reference)
			if !ok {
				continue
			}

			if !policyExists(res, policyReference.ID()) {
				return fmt.Errorf("group '%s' environment '%s' references missing policy '%s'", group.Name, env.Name, policyReference.ID())
			}
		}
	}
	if group.Account != nil {
		// check references in account policies
		for _, policyRef := range group.Account.Policies {
			policyReference, ok := policyRef.(account.Reference)
			if !ok {
				continue
			}

			if !policyExists(res, policyReference.ID()) {
				return fmt.Errorf("group '%s' account references missing policy '%s'", group.Name, policyReference.ID())
			}
		}
	}
	return nil
}

func groupExists(a *account.Resources, id string) bool {
	_, exists := a.Groups[id]
	return exists
}

func policyExists(a *account.Resources, id string) bool {
	_, exists := a.Policies[id]
	return exists

}

func validateFile(file types.File) error {
	for _, p := range file.Policies {
		if err := validatePolicy(p); err != nil {
			return err
		}
	}

	for _, g := range file.Groups {
		if err := validateGroup(g); err != nil {
			return err
		}
	}

	for _, u := range file.Users {
		if err := validateUser(u); err != nil {
			return err
		}
	}

	if featureflags.ServiceUsers.Enabled() {
		for _, su := range file.ServiceUsers {
			if err := validateServiceUser(su); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateUser(u types.User) error {
	if u.Email == "" {
		return errors.New("missing required field 'email' for user")
	}

	for _, groupRef := range u.Groups {
		if err := validateReference(groupRef); err != nil {
			return err
		}
	}
	return nil
}

func validateServiceUser(su types.ServiceUser) error {
	if su.Name == "" {
		return errors.New("missing required field 'name' for service user")
	}

	for _, groupRef := range su.Groups {
		if err := validateReference(groupRef); err != nil {
			return err
		}
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

	for _, env := range g.Environment {
		for _, policyRef := range env.Policies {
			if err := validateReference(policyRef); err != nil {
				return err
			}
		}
	}

	if g.Account != nil {
		// check references in account policies
		for _, policyRef := range g.Account.Policies {
			if err := validateReference(policyRef); err != nil {
				return err
			}
		}
	}

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

func validateReference(reference types.Reference) error {
	if reference.Type == types.ReferenceType {
		if reference.Id == "" {
			return errors.New("missing required field 'id' for reference")
		}
	} else if reference.Value == "" {
		return errors.New("missing reference value")
	}
	return nil
}
