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
		Policies []Policy `yaml:"policies,omitempty"`
		Groups   []Group  `yaml:"groups,omitempty"`
		Users    []User   `yaml:"users,omitempty"`
	}
	Policy struct {
		ID             string      `yaml:"id"`
		Name           string      `yaml:"name"`
		Level          PolicyLevel `yaml:"level"`
		Description    string      `yaml:"description,omitempty"`
		Policy         string      `yaml:"policy"`
		OriginObjectID string      `yaml:"originObjectId,omitempty"`
	}
	PolicyLevel struct {
		Type        string `yaml:"type"`
		Environment string `yaml:"environment,omitempty"`
	}
	Group struct {
		ID             string           `yaml:"id"`
		Name           string           `yaml:"name"`
		Description    string           `yaml:"description,omitempty"`
		Account        *Account         `yaml:"account,omitempty"`
		Environment    []Environment    `yaml:"environment,omitempty"`
		ManagementZone []ManagementZone `yaml:"managementZone,omitempty"`
		OriginObjectID string           `yaml:"originObjectId,omitempty"`
	}
	Account struct {
		Permissions []string    `yaml:"permissions,omitempty"`
		Policies    []Reference `yaml:"policies,omitempty"`
	}
	Environment struct {
		Name        string      `yaml:"name"`
		Permissions []string    `yaml:"permissions,omitempty"`
		Policies    []Reference `yaml:"policies,omitempty"`
	}
	ManagementZone struct {
		Environment    string   `yaml:"environment"`
		ManagementZone string   `yaml:"managementZone"`
		Permissions    []string `yaml:"permissions"`
	}
	User struct {
		Email  string      `yaml:"email"`
		Groups []Reference `yaml:"groups,omitempty"`
	}

	Reference struct {
		Type  string `yaml:"type" mapstructure:"type"`
		Id    string `yaml:"id" mapstructure:"id"`
		Value string `yaml:"-" mapstructure:"-"` // omitted from being written/read
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

const (
	KeyUsers    string = "users"
	KeyGroups   string = "groups"
	KeyPolicies string = "policies"
)
