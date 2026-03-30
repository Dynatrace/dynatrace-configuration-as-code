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

package types

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
)

const (
	ReferenceType          = "reference"
	PolicyLevelAccount     = "account"
	PolicyLevelEnvironment = "environment"
)

type (
	File struct {
		Boundaries   []Boundary    `yaml:"boundaries,omitempty" json:"boundaries,omitempty"`
		Policies     []Policy      `yaml:"policies,omitempty" json:"policies,omitempty"`
		Groups       []Group       `yaml:"groups,omitempty" json:"groups,omitempty"`
		Users        []User        `yaml:"users,omitempty" json:"users,omitempty"`
		ServiceUsers []ServiceUser `yaml:"serviceUsers,omitempty" json:"serviceUsers,omitempty"`
	}

	Boundary struct {
		ID             string `yaml:"id" json:"id"`
		Name           string `yaml:"name" json:"name"`
		Query          string `yaml:"query" json:"query"`
		OriginObjectID string `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}

	Policy struct {
		ID             string      `yaml:"id" json:"id"`
		Name           string      `yaml:"name" json:"name"`
		Level          PolicyLevel `yaml:"level" json:"level"`
		Description    string      `yaml:"description,omitempty" json:"description,omitempty"`
		Policy         string      `yaml:"policy" json:"policy"`
		OriginObjectID string      `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}

	PolicyLevel struct {
		Type        string `yaml:"type" json:"type"`
		Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
	}

	Group struct {
		ID                       string   `yaml:"id" json:"id"`
		Name                     string   `yaml:"name" json:"name"`
		Description              string   `yaml:"description,omitempty" json:"description,omitempty"`
		FederatedAttributeValues []string `yaml:"federatedAttributeValues,omitempty" json:"federatedAttributeValues,omitempty"`
		// Account level permissions and policies that apply to users in this group
		Account *Account `yaml:"account,omitempty" json:"account,omitempty"`
		// Environment level permissions and policies that apply to users in this group
		Environment []Environment `yaml:"environments,omitempty" json:"environments,omitempty"`
		// ManagementZone level permissions that apply to users in this group
		ManagementZone []ManagementZone `yaml:"managementZones,omitempty" json:"managementZones,omitempty"`
		OriginObjectID string           `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}

	Account struct {
		Permissions []string        `yaml:"permissions,omitempty" json:"permissions,omitempty"`
		Policies    []PolicyBinding `yaml:"policies,omitempty" json:"policies,omitempty"`
	}

	Environment struct {
		Name        string          `yaml:"environment" json:"environment"`
		Permissions []string        `yaml:"permissions,omitempty" json:"permissions,omitempty"`
		Policies    []PolicyBinding `yaml:"policies,omitempty" json:"policies,omitempty"`
	}

	ManagementZone struct {
		Environment    string   `yaml:"environment" json:"environment"`
		ManagementZone string   `yaml:"managementZone" json:"managementZone"`
		Permissions    []string `yaml:"permissions" json:"permissions"`
	}

	User struct {
		Email  secret.Email   `yaml:"email" json:"email"`
		Groups ReferenceSlice `yaml:"groups,omitempty" json:"groups,omitempty"`
	}

	ServiceUser struct {
		Name           string         `yaml:"name" json:"name"`
		Description    string         `yaml:"description,omitempty" json:"description,omitempty"`
		Groups         ReferenceSlice `yaml:"groups,omitempty" json:"groups,omitempty"`
		OriginObjectID string         `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}

	Reference struct {
		Type  string `yaml:"type" json:"type" mapstructure:"type"`
		Id    string `yaml:"id" json:"id" mapstructure:"id"`
		Value string `yaml:"-" json:"-" mapstructure:"-"` // omitted from being written/read
	}

	PolicyBinding struct {
		Type       string         `yaml:"type,omitempty" json:"type,omitempty" mapstructure:"type"` // shorthand syntax for backwards compatibility
		Id         string         `yaml:"id,omitempty" json:"id,omitempty" mapstructure:"id"`       // shorthand syntax for backwards compatibility
		Value      string         `yaml:"-" json:"-" mapstructure:"-"`                              // omitted from being written/read // shorthand syntax for backwards compatibility
		Policy     *Reference     `yaml:"policy,omitempty" json:"policy,omitempty" mapstructure:"policy"`
		Boundaries ReferenceSlice `yaml:"boundaries,omitempty" json:"boundaries,omitempty" mapstructure:"boundaries"`
	}
)

// UnmarshalYAML is a custom yaml.Unmarshaler for Reference able to parse simple string values and actual references.
// As it unmarshalls data into the Reference r, it has a pointer receiver.
func (r *Reference) UnmarshalYAML(unmarshal func(any) error) error {
	var data any
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch data := data.(type) {
	case string:
		r.Value = data
	default:
		if err := mapstructure.Decode(data, &r); err != nil {
			return fmt.Errorf("failed to parse reference: %w", err)
		}
	}
	return nil
}

// UnmarshalYAML is a custom yaml.Unmarshaler for PolicyBinding able to parse simple string values and references.
// As it unmarshalls data into the PolicyBinding r, it has a pointer receiver.
func (r *PolicyBinding) UnmarshalYAML(unmarshal func(any) error) error {
	// We first try to unmarshal the YAML value as a string to supported shorthand syntax like "policy: My policy name"
	var data string
	if err := unmarshal(&data); err == nil {
		r.Value = data
		return nil
	}

	// The shorthand syntax unmarshalling did not work, so it must be the PolicyBinding struct.
	// A temporary struct is defined. We cannot just use "any" as then the UnmarshalYAML function for Reference would not get called.
	type policyBinding PolicyBinding
	var temp policyBinding

	if err := unmarshal(&temp); err != nil {
		return err
	}
	*r = PolicyBinding(temp)
	return nil
}

// MarshalYAML is a custom yaml.Marshaler for PolicyBinding, able to write simple string values and actual references.
// As it is called when marshalling PolicyBinding values, it has a value receiver.
func (r PolicyBinding) MarshalYAML() (any, error) {
	if r.Policy == nil {
		if r.Type == ReferenceType {
			return r, nil
		}

		// if not a reference, just marshal the value string
		return r.Value, nil
	}

	return r, nil
}

// MarshalYAML is a custom yaml.Marshaler for Reference, able to write simple string values and actual references.
// As it is called when marshalling Reference values, it has a value receiver.
func (r Reference) MarshalYAML() (any, error) {
	if r.Type == ReferenceType {
		return r, nil
	}

	// if not a reference, just marshal the value string
	return r.Value, nil
}

type ReferenceSlice []Reference

const (
	KeyUsers        string = "users"
	KeyServiceUsers string = "serviceUsers"
	KeyGroups       string = "groups"
	KeyPolicies     string = "policies"
	KeyBoundaries   string = "boundaries"
)
