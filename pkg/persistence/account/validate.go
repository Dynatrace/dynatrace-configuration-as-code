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

package account

import "fmt"

// Validate checks the references in the provided AMResources instance to ensure
// that all referenced groups and policies exist. It iterates through the users,
// environment policies, and account policies, validating their references.
func Validate(res *AMResources) error {
	for _, user := range res.Users {
		for _, groupRef := range user.Groups {
			if err := refCheck(groupRef, res.GroupExists); err != nil {
				return err
			}
		}
	}

	for _, group := range res.Groups {
		// check references in environment policies
		for _, env := range group.Environment {
			for _, policyRef := range env.Policies {
				if err := refCheck(policyRef, res.PolicyExists); err != nil {
					return err
				}
			}
		}
		// check references in account policies
		for _, policyRef := range group.Account.Policies {
			if err := refCheck(policyRef, res.PolicyExists); err != nil {
				return err
			}
		}
	}
	return nil
}

func refCheck(elem any, refCheckFn func(string) bool) error {
	if reference, isCustomRef := elem.(anyMap); isCustomRef {
		idField, hasIdField := reference["id"]
		if !hasIdField {
			return fmt.Errorf("error validating account resources: %w", ErrIdFieldMissing)
		}
		idFieldStr, isIdFieldStr := idField.(string)
		if !isIdFieldStr {
			return fmt.Errorf("error validating account resources: %w", ErrIdFieldNoString)
		}
		policyRefExists := refCheckFn(idFieldStr)
		if !policyRefExists {
			return fmt.Errorf("error validating account resources with id %q: %w", idFieldStr, ErrRefMissing)
		}
	}
	return nil
}

func (a *AMResources) GroupExists(id string) bool {
	for _, g := range a.Groups {
		if g.ID == id {
			return true
		}
	}
	return false
}

func (a *AMResources) PolicyExists(id string) bool {
	for _, p := range a.Policies {
		if p.ID == id {
			return true
		}
	}
	return false
}
