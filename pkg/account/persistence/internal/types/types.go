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

	"github.com/invopop/jsonschema"
	"github.com/mitchellh/mapstructure"

	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
)

const (
	ReferenceType          = "reference"
	PolicyLevelAccount     = "account"
	PolicyLevelEnvironment = "environment"
)

type (
	File struct {
		Boundaries   []Boundary    `yaml:"boundaries,omitempty" json:"boundaries,omitempty" jsonschema:"description=Boundaries to configure for this account."`
		Policies     []Policy      `yaml:"policies,omitempty" json:"policies,omitempty" jsonschema:"description=Policies to configure for this account."`
		Groups       []Group       `yaml:"groups,omitempty" json:"groups,omitempty" jsonschema:"description=Groups to configure for this account."`
		Users        []User        `yaml:"users,omitempty" json:"users,omitempty" jsonschema:"description=Users to configure for this account."`
		ServiceUsers []ServiceUser `yaml:"serviceUsers,omitempty" json:"serviceUsers,omitempty" jsonschema:"description=Service users to configure for this account."`
	}

	Boundary struct {
		ID             string `yaml:"id" json:"id" jsonschema:"required,description=A unique identifier of this boundary configuration - this can be freely defined, used by monaco."`
		Name           string `yaml:"name" json:"name" jsonschema:"required,description=The name of this boundary."`
		Query          string `yaml:"query" json:"query" jsonschema:"required,description=The query definition of the boundary."`
		OriginObjectID string `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty" jsonschema:"description=The identifier of the boundary this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object."`
	}

	Policy struct {
		ID             string      `yaml:"id" json:"id" jsonschema:"required,description=A unique identifier of this policy configuration - this can be freely defined, used by monaco."`
		Name           string      `yaml:"name" json:"name" jsonschema:"required,description=The name of this policy."`
		Level          PolicyLevel `yaml:"level" json:"level" jsonschema:"required,description=The level this policy applies to."`
		Description    string      `yaml:"description,omitempty" json:"description,omitempty" jsonschema:"A description of this policy."`
		Policy         string      `yaml:"policy" json:"policy" jsonschema:"required,description=The policy definition."`
		OriginObjectID string      `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty" jsonschema:"description=The identifier of the policy this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object."`
	}

	PolicyLevel struct {
		Type        string `yaml:"type" json:"type" jsonschema:"required,enum=account,enum=environment,description=This defines which level this policy applies to - either the whole 'account' or a specific 'environment'. For environment level, the 'environment' field needs to contain the environment ID."`
		Environment string `yaml:"environment,omitempty" json:"environment,omitempty" jsonschema:"The ID of the environment this policy applies to. Required if type is 'environment'."`
	}

	Group struct {
		ID                       string   `yaml:"id" json:"id" jsonschema:"required,description=A unique identifier of this group configuration - this can be freely defined, used by monaco."`
		Name                     string   `yaml:"name" json:"name" jsonschema:"required,description=The name of this group."`
		Description              string   `yaml:"description,omitempty" json:"description,omitempty" jsonschema:"A description of this group."`
		FederatedAttributeValues []string `yaml:"federatedAttributeValues,omitempty" json:"federatedAttributeValues,omitempty" jsonschema:"Federated attribute values of this group."`
		// Account level permissions and policies that apply to users in this group
		Account *Account `yaml:"account,omitempty" json:"account,omitempty" jsonschema:"description=Account level permissions and policies that apply to users in this group."`
		// Environment level permissions and policies that apply to users in this group
		Environment []Environment `yaml:"environments,omitempty" json:"environments,omitempty" jsonschema:"description=Environment level permissions and policies that apply to users in this group."`
		// ManagementZone level permissions that apply to users in this group
		ManagementZone []ManagementZone `yaml:"managementZones,omitempty" json:"managementZones,omitempty" jsonschema:"description=ManagementZone level permissions that apply to users in this group."`
		OriginObjectID string           `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty" jsonschema:"description=The identifier of the group this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object."`
	}

	Account struct {
		Permissions []string       `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Permissions for the whole account."`
		Policies    ReferenceSlice `yaml:"policies,omitempty" json:"policies,omitempty" jsonschema:"description=Policies for the whole account."`
	}

	Environment struct {
		Name        string         `yaml:"environment" json:"environment" jsonschema:"required,description=Name/identifier of the environment."`
		Permissions []string       `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Permissions for this environment."`
		Policies    ReferenceSlice `yaml:"policies,omitempty" json:"policies,omitempty" jsonschema:"description=Policies for this environment."`
	}

	ManagementZone struct {
		Environment    string   `yaml:"environment" json:"environment" jsonschema:"required,description=Name/identifier of the environment the management zone is in."`
		ManagementZone string   `yaml:"managementZone" json:"managementZone" jsonschema:"required,description=Identifier of the management zone."`
		Permissions    []string `yaml:"permissions" json:"permissions" jsonschema:"required,description=Permissions for this management zone."`
	}

	User struct {
		Email  secret.Email   `yaml:"email" json:"email" jsonschema:"required,description=Email address of this user."`
		Groups ReferenceSlice `yaml:"groups,omitempty" json:"groups,omitempty" jsonschema:"description=Groups this user is part of - either defined by name directly or as a reference to a group configuration."`
	}

	ServiceUser struct {
		Name           string         `yaml:"name" json:"name" jsonschema:"required,description=The name of this service user."`
		Description    string         `yaml:"description,omitempty" json:"description,omitempty" jsonschema:"A description of this service user."`
		Groups         ReferenceSlice `yaml:"groups,omitempty" json:"groups,omitempty" jsonschema:"description=Groups this user is part of - either defined by name directly or as a reference to a group configuration."`
		OriginObjectID string         `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty" jsonschema:"description=The identifier of the service user this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object."`
	}

	Reference struct {
		Type  string `yaml:"type" json:"type" mapstructure:"type" jsonschema:"enum=reference"`
		Id    string `yaml:"id" json:"id" mapstructure:"id" jsonschema:"description=The 'id' of the account configuration being referenced."`
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
func (ReferenceSlice) JSONSchema() *jsonschema.Schema {
	base := jsonutils.ReflectJSONSchema(Reference{})

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
			Properties:           base.Properties,
			AdditionalProperties: base.AdditionalProperties,
			Required:             base.Required,
			Comments:             base.Comments,
		},
	}
}

const (
	KeyUsers        string = "users"
	KeyServiceUsers string = "serviceUsers"
	KeyGroups       string = "groups"
	KeyPolicies     string = "policies"
	KeyBoundaries   string = "boundaries"
)
