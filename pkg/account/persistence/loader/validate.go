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

func validatePolicyBindingReferences(res *account.Resources, policies []account.PolicyBinding) error {
	for _, policyRef := range policies {
		policyReference, ok := policyRef.Policy.(account.Reference)
		if !ok {
			continue
		}

		if !policyExists(res, policyReference.ID()) {
			return fmt.Errorf("missing policy '%s'", policyReference.ID())
		}

		for _, boundaryRef := range policyRef.Boundaries {
			boundaryReference, ok := boundaryRef.(account.Reference)
			if !ok {
				continue
			}

			if !boundaryExists(res, boundaryReference.ID()) {
				return fmt.Errorf("policy '%s' references missing boundary '%s'", policyRef.Policy.ID(), boundaryReference.ID())
			}
		}
	}

	return nil
}

func validateGroupReferences(res *account.Resources, group account.Group) error {
	// check policy bindings in environment policies
	for _, env := range group.Environment {
		if err := validatePolicyBindingReferences(res, env.Policies); err != nil {
			return fmt.Errorf("group '%s' has an invalid policy reference for environment '%s': %w", group.Name, env.Name, err)
		}
	}
	if group.Account != nil {
		// check policy bindings in account policies
		if err := validatePolicyBindingReferences(res, group.Account.Policies); err != nil {
			return fmt.Errorf("group '%s' has an invalid account policy reference: %w", group.Name, err)
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

func boundaryExists(a *account.Resources, id string) bool {
	_, exists := a.Boundaries[id]
	return exists
}

func validateFile(file types.File) error {
	for _, b := range file.Boundaries {
		if err := validateBoundary(b); err != nil {
			return err
		}
	}

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

	for _, su := range file.ServiceUsers {
		if err := validateServiceUser(su); err != nil {
			return err
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
		if err := validateGroupEnvironment(env, g.ID); err != nil {
			return err
		}
	}

	if g.Account != nil {
		// check references in account policies
		for _, policyRef := range g.Account.Policies {
			if err := validatePolicyBinding(policyRef); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateGroupEnvironment(env types.Environment, groupID string) error {
	if env.Name == "" {
		return fmt.Errorf("missing required field 'environment' for 'environments' in group '%s'", groupID)
	}
	for _, policyRef := range env.Policies {
		if err := validatePolicyBinding(policyRef); err != nil {
			return err
		}
	}
	return nil
}

func validateBoundary(b types.Boundary) error {
	if b.ID == "" {
		return errors.New("missing required field 'id' for boundary")
	}
	if b.Name == "" {
		return fmt.Errorf("missing required field 'name' for boundary %q", b.ID)
	}
	if b.Query == "" {
		return fmt.Errorf("missing required field 'query' for boundary %q", b.ID)
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

func validatePolicyBinding(policyBinding types.PolicyBinding) error {
	// We don't need to check for the Value field. The Value field is only set when no key is used, i.e.
	// policies:
	//   - My policy
	// In such a case it would be an invalid yaml file anyway if a child element "policy" was to be added.

	if policyBinding.Policy == nil {
		return validateLegacyPolicyBinding(policyBinding)
	}

	if policyBinding.Id != "" {
		return fmt.Errorf("policy definition is ambiguous. Use the 'policy' key only and remove the keys 'id' and 'type' for policy with id '%v'", policyBinding.Id)
	}

	if err := validateReference(*policyBinding.Policy); err != nil {
		return err
	}

	for _, boundaryRef := range policyBinding.Boundaries {
		if err := validateReference(boundaryRef); err != nil {
			return err
		}
	}

	return nil
}

func validateLegacyPolicyBinding(policyBinding types.PolicyBinding) error {
	if policyBinding.Type == types.ReferenceType {
		if policyBinding.Id == "" {
			return errors.New("missing required field 'id' for reference")
		}
	} else if policyBinding.Value == "" {
		return errors.New("missing reference value")
	}

	if len(policyBinding.Boundaries) > 0 {
		return fmt.Errorf("error when loading boundary reference with id %v: boundaries are only supported when using the 'policy' key", policyBinding.Id)
	}
	return nil
}
