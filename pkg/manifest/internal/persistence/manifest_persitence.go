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

package persistence

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

const SimpleProjectType = "simple"
const GroupProjectType = "grouping"

type Project struct {
	Name string `yaml:"name" json:"name" jsonschema:"description=..."`
	Type string `yaml:"type,omitempty" json:"type" jsonschema:"description=..."`
	Path string `yaml:"path,omitempty" json:"path" jsonschema:"description=..."`
}

type Type string

const (
	TypeEnvironment Type = "environment"
	TypeValue       Type = "value"
)

// TypedValue represents a value with a Type - currently these are variables that can be either:
// - TypeEnvironment...loaded from an environment variable
// - TypeValue...read directly
// Additionally TypedValues can be defined directly as a string, as a shorthand for type: TypeValue
type TypedValue struct {
	Type  Type   `yaml:"type,omitempty" mapstructure:"type" json:"type" jsonschema:"description=..."`
	Value string `yaml:"value" mapstructure:"value" json:"name" jsonschema:"description=..."`
}

// UnmarshalYAML Custom unmarshaler for TypedValue able to parse simple shorthands (accountUUID: 1234) and full values.
func (c *TypedValue) UnmarshalYAML(unmarshal func(any) error) error {
	var data any
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch data.(type) {
	case string:
		c.Type = TypeValue
		c.Value = data.(string)
	default:
		if err := mapstructure.Decode(data, c); err != nil {
			return fmt.Errorf("failed to parse accountUUID: %w", err)
		}
	}
	return nil
}

// AuthSecret represents a user-defined client id or client secret. It has a [Type] which is [TypeEnvironment] (default).
// Secrets must never be provided as plain text, but always loaded from somewhere else. Currently, loading is only allowed from environment variables.
//
// [Name] contains the environment-variable to resolve the authSecret.
//
// This struct is meant to be reused for fields that require the same behavior.
type AuthSecret struct {
	Type Type   `yaml:"type"`
	Name string `yaml:"name"`
}

type OAuth struct {
	ClientID      AuthSecret  `yaml:"clientId" json:"clientId" jsonschema:"description=..."`
	ClientSecret  AuthSecret  `yaml:"clientSecret" json:"clientSecret" jsonschema:"description=..."`
	TokenEndpoint *TypedValue `yaml:"tokenEndpoint,omitempty" json:"tokenEndpoint" jsonschema:"description=..."`
}

type Auth struct {
	Token AuthSecret `yaml:"token" json:"token" jsonschema:"description=..."`
	OAuth *OAuth     `yaml:"oAuth,omitempty" json:"oAuth" jsonschema:"description=..."`
}

type Environment struct {
	Name string     `yaml:"name"  json:"name" jsonschema:"description=..."`
	URL  TypedValue `yaml:"url" json:"url" jsonschema:"description=..."`

	// Auth contains all authentication related information
	Auth Auth `yaml:"auth,omitempty" json:"auth" jsonschema:"description=..."`
}

type Group struct {
	Name         string        `yaml:"name" json:"name" jsonschema:"description=..."`
	Environments []Environment `yaml:"environments" json:"environments" jsonschema:"description=..."`
}

type Manifest struct {
	ManifestVersion   string    `yaml:"manifestVersion" json:"name" jsonschema:"description=..."`
	Projects          []Project `yaml:"projects" json:"projects" jsonschema:"description=..."`
	EnvironmentGroups []Group   `yaml:"environmentGroups" json:"environmentGroups" jsonschema:"description=..."`
	Accounts          []Account `yaml:"accounts,omitempty" json:"accounts" jsonschema:"description=..."`
}

type Account struct {
	Name        string      `yaml:"name" json:"name" jsonschema:"description=..."`
	AccountUUID TypedValue  `yaml:"accountUUID" json:"accountUUID" jsonschema:"description=..."`
	ApiUrl      *TypedValue `yaml:"apiUrl,omitempty" json:"apiUrl" jsonschema:"description=..."`
	OAuth       OAuth       `yaml:"oAuth" json:"oAuth" jsonschema:"description=..."`
}
