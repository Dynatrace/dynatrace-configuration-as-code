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
	Name string `yaml:"name" json:"name" jsonschema:"required"`
	Type string `yaml:"type,omitempty" json:"type" jsonschema:"optional,enum=simple,enum=grouping"`
	Path string `yaml:"path,omitempty" json:"path" jsonschema:"optional"`
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
	// Type of this value either an 'environment' variable to read, or simpy a 'value' directly in the YAML.
	Type Type `yaml:"type,omitempty" mapstructure:"type" json:"type" jsonschema:"enum=environment,enum=value"`
	// Value, depending on Type either the name of an environment variable or just a string value.
	Value string `yaml:"value" mapstructure:"value" json:"value" jsonschema:"required"`
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
	// Type exists for future compatibility - AuthSecrets are always read from 'environment' variables
	Type Type `yaml:"type" json:"type,omitempty"`
	//Name of the environment variable to read the secret from.
	Name string `yaml:"name" json:"name" jsonschema:"required"`
}

// OAuth defines the required information to request oAuth bearer tokens for authenticated API calls
type OAuth struct {
	// ClientID of the oAuth client credentials used to request bearer-tokens for authenticated API calls
	ClientID AuthSecret `yaml:"clientId" json:"clientId" jsonschema:"required"`
	// ClientSecret of the oAuth client credentials used to request bearer-tokens for authenticated API calls
	ClientSecret AuthSecret `yaml:"clientSecret" json:"clientSecret" jsonschema:"required"`
	// TokenEndpoint allows to optionally define a non-standard endpoint to request bearer-tokens from. default: sso.dynatrace.com
	TokenEndpoint *TypedValue `yaml:"tokenEndpoint,omitempty" json:"tokenEndpoint" jsonschema:"oneof_type=string;object"`
}

// Auth defines all required information for authenticated API calls
type Auth struct {
	// Token defines an API access tokens used for Dynatrace Config API calls
	Token AuthSecret `yaml:"token" json:"token" jsonschema:"required"`
	// OAuth defines client credentials used for Dynatrace Platform API calls - for platform environments this is required
	OAuth *OAuth `yaml:"oAuth,omitempty" json:"oAuth"`
}

// Environment defines all required information for accessing a Dynatrace environment
type Environment struct {
	// Name of the environment - this can be freely defined and will be used in logs, etc.
	Name string `yaml:"name"  json:"name" jsonschema:"required"`
	// URL of the environment
	URL TypedValue `yaml:"url" json:"url" jsonschema:"required,oneof_type=string;object"`

	// Auth contains all information required for authenticated access to the environment's API
	Auth Auth `yaml:"auth,omitempty" json:"auth" jsonschema:"required"`
}

// Group defines a group of Environment
type Group struct {
	// Name of the group - this can be freely defined and will be used in logs, etc.
	Name string `yaml:"name" json:"name" jsonschema:"required"`
	// Environments that are part of this group
	Environments []Environment `yaml:"environments" json:"environments" jsonschema:"required,minLength=1"`
}

type Manifest struct {
	// ManifestVersion is the version of this manifest. It is used when loading a manifest to ensure the CLI version is able to parse this manifest
	ManifestVersion string `yaml:"manifestVersion" json:"manifestVersion"  jsonschema:"required,oneof_type=string;number"`
	// Projects is a list of projects that will be deployed with this manifest
	Projects []Project `yaml:"projects" json:"projects" jsonschema:"required,minLength=1"`
	// EnvironmentGroups is a list of environment groups that configs in Projects will be deployed to
	EnvironmentGroups []Group `yaml:"environmentGroups" json:"environmentGroups" jsonschema:"minLength=1"`
	// EnvironmentGroups is a list of accounts that account resources in Projects will be deployed to
	Accounts []Account `yaml:"accounts,omitempty" json:"accounts" jsonschema:"minLength=1"`
}

type Account struct {
	// Name of the account - this can be freely defined and will show up in logs, etc.
	Name string `yaml:"name" json:"name" jsonschema:"type=string"`
	// AccountUUID is the uuid of your account - you can find this in myaccount.
	AccountUUID TypedValue `yaml:"accountUUID" json:"accountUUID" jsonschema:"required,oneof_type=string;object"`
	// ApiUrl allows to optionally define a different Account Management API URL. Default: api.dynatrace.com
	ApiUrl *TypedValue `yaml:"apiUrl,omitempty" json:"apiUrl" jsonschema:"optional,oneof_type=string;object"`
	// OAuth client credentials to authenticate API calls for this account.
	OAuth OAuth `yaml:"oAuth" json:"oAuth" jsonschema:"required"`
}
