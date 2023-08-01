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
	"github.com/iancoleman/orderedmap"
	"github.com/invopop/jsonschema"
	"github.com/mitchellh/mapstructure"
)

const (
	ReferenceType          = "reference"
	PolicyLevelAccount     = "account"
	PolicyLevelEnvironment = "environment"
)

type (
	Resources struct {
		Policies map[string]Policy
		Groups   map[string]Group
		Users    map[string]User
	}
	File struct {
		// Policies to configure for this account
		Policies []Policy `yaml:"policies,omitempty" json:"policies,omitempty"`
		// Groups to configure for this account
		Groups []Group `yaml:"groups,omitempty" json:"groups,omitempty"`
		// Users to configure for this account
		Users []User `yaml:"users,omitempty" json:"users,omitempty"`
	}
	Policy struct {
		// ID of this policy configuration, used by monaco
		ID string `yaml:"id" json:"id" jsonschema:"required"`
		// Name of this policy
		Name string `yaml:"name" json:"name" jsonschema:"required"`
		// Level this policy applies to
		Level PolicyLevel `yaml:"level" json:"level" jsonschema:"required"`
		// Description for this policy
		Description string `yaml:"description,omitempty" json:"description,omitempty"`
		// Policy string
		Policy string `yaml:"policy" json:"policy" jsonschema:"required"`
		// OriginObjectID defines the identifier of the policy this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object
		OriginObjectID string `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}
	PolicyLevel struct {
		// Type defines what level this policy applies to - either the whole 'account' or a specific 'environment'. For environment level, the Environment field needs to contain the environment ID
		Type string `yaml:"type" json:"type" jsonschema:"required,enum=account,enum=environment"`
		// Environment ID this policy applies to. Required if type is 'environment'
		Environment string `yaml:"environment,omitempty" json:"environment,omitempty"`
	}
	Group struct {
		// ID of this group configuration, used by monaco
		ID string `yaml:"id" json:"id" jsonschema:"required"`
		// Name of this group
		Name string `yaml:"name" json:"name" jsonschema:"required"`
		// Description for this group
		Description string `yaml:"description,omitempty" json:"description,omitempty"`
		// Account level permissions and policies that apply to users in this group
		Account *Account `yaml:"account,omitempty" json:"account,omitempty"`
		// Environment level permissions and policies that apply to users in this group
		Environment []Environment `yaml:"environment,omitempty" json:"environment,omitempty"`
		// ManagementZone level permissions that apply to users in this group
		ManagementZone []ManagementZone `yaml:"managementZone,omitempty" json:"managementZone,omitempty"`
		// OriginObjectID defines the identifier of the group this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object
		OriginObjectID string `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
	}
	Account struct {
		// Permissions for the whole account
		Permissions []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`
		// Policies for the whole account
		Policies ReferenceSlice `yaml:"policies,omitempty" json:"policies,omitempty"`
	}
	Environment struct {
		// Name identifier of the environment
		Name string `yaml:"name" json:"name" jsonschema:"required"`
		// Permissions for this environment
		Permissions []string `yaml:"permissions,omitempty" json:"permissions,omitempty"`
		// Policies for this environment
		Policies ReferenceSlice `yaml:"policies,omitempty" json:"policies,omitempty"`
	}
	ManagementZone struct {
		// Environment identifier of the environment
		Environment string `yaml:"environment" json:"environment" jsonschema:"required"`
		// ManagementZone identifier
		ManagementZone string `yaml:"managementZone" json:"managementZone" jsonschema:"required"`
		// Permissions for this ManagementZone in the Environment
		Permissions []string `yaml:"permissions" json:"permissions" jsonschema:"required"`
	}
	User struct {
		// Email address of this user
		Email string `yaml:"email" json:"email" jsonschema:"required"`
		// Groups this user is part of - either defined by name or as a Reference object
		Groups ReferenceSlice `yaml:"groups,omitempty" json:"groups,omitempty"`
	}

	Reference struct {
		// Type 'reference'
		Type string `yaml:"type" json:"type" mapstructure:"type" jsonschema:"enum=reference"`
		// Id of the account configuration to reference
		Id    string `yaml:"id" json:"id" mapstructure:"id"`
		Value string `yaml:"-" json:"-" mapstructure:"-"` // omitted from being written/read
	}
)

// UnmarshalYAML is a custom yaml.Unmarshaler for Reference able to parse simple string values and actual references.
// As it unmarshalls data into the Reference r, it has a pointer receiver.
func (r *Reference) UnmarshalYAML(unmarshal func(any) error) error {
	var data any
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch data.(type) {
	case string:
		r.Value = data.(string)
	default:
		if err := mapstructure.Decode(data, &r); err != nil {
			return fmt.Errorf("failed to parse reference: %w", err)
		}
	}
	return nil
}

// MarshalYAML is a custom yaml.Marshaler for Reference, able to write simple string values and actual references.
// As it is called when marshalling Reference values, it has a value receiver.
func (r Reference) MarshalYAML() (interface{}, error) {
	if r.Type == ReferenceType {
		return r, nil
	}

	// if not a reference, just marshal the value string
	return r.Value, nil
}

type ReferenceSlice []Reference

// JSONSchema defines a custom schema definition for ReferenceSlice as it contains either Reference objects or strings
// when being parsed, but our schema generator can not resolve such a nested "one-of" relation correctly for slices
func (r ReferenceSlice) JSONSchema() *jsonschema.Schema {
	props := orderedmap.New()
	props.Set("type", map[string]any{"type": "string", "enum": []string{"reference"}, "description": "Type 'reference'"})
	props.Set("id", map[string]any{"type": "string", "description": "Id of the account configuration to reference"})

	return &jsonschema.Schema{
		Type: "array",
		Items: &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				{
					Type: "string",
				},
				{
					Type: "object",
				},
			},
			Properties:           props,
			AdditionalProperties: jsonschema.FalseSchema,
			Required:             []string{"type", "id"},
		},
	}
}

const (
	KeyUsers    string = "users"
	KeyGroups   string = "groups"
	KeyPolicies string = "policies"
)
