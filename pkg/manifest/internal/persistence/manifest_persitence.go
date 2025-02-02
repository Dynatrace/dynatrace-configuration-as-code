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
	Name string `yaml:"name" json:"name" jsonschema:"required,description=The name of the project - if 'path' is not set the name will be used as path, otherwise this can be freely defined."`
	Type string `yaml:"type,omitempty" json:"type" jsonschema:"enum=simple,enum=grouping,description=The type of project - either a 'simple' project folder containing configs, or a 'grouping' of projects in sub-folders."`
	Path string `yaml:"path,omitempty" json:"path" jsonschema:"description=The file path to the project folder, relative to the manifest's location."`
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
	Type  Type   `yaml:"type,omitempty" mapstructure:"type" json:"type" jsonschema:"enum=environment,enum=value,description=The type of this value - either an 'environment' variable to read, or simpy a 'value' directly in the YAML."`
	Value string `yaml:"value" mapstructure:"value" json:"value" jsonschema:"required,description=The value is depending on 'type' either the name of an environment variable to load or just a string value."`
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
	ClientID AuthSecret `yaml:"clientId" json:"clientId" jsonschema:"required,description=The ID of the oAuth client credentials used to request bearer-tokens for authenticated API calls."`
	// ClientSecret of the oAuth client credentials used to request bearer-tokens for authenticated API calls
	ClientSecret AuthSecret `yaml:"clientSecret" json:"clientSecret" jsonschema:"required,description=The secret of the oAuth client credentials used to request bearer-tokens for authenticated API calls."`
	// TokenEndpoint allows to optionally define a non-standard endpoint to request bearer-tokens from. Defaults to production sso.dynatrace.com if not defined.
	TokenEndpoint *TypedValue `yaml:"tokenEndpoint,omitempty" json:"tokenEndpoint" jsonschema:"oneof_type=string;object,default=sso.dynatrace.com,description=This allows to optionally define a non-standard endpoint to request bearer tokens from."`
}

// Auth defines all required information for authenticated API calls
type Auth struct {
	// Token defines an API access tokens used for Dynatrace Config API calls
	Token *AuthSecret `yaml:"token,omitempty" json:"token" jsonschema:"description=An API access tokens used for Dynatrace Config API calls - for classic apis this is required"`
	// OAuth defines client credentials used for Dynatrace Platform API calls
	OAuth *OAuth `yaml:"oAuth,omitempty" json:"oAuth" jsonschema:"description=OAuth client credentials used for Dynatrace Platform API calls - for platform environments this is required."`
}

// Environment defines all required information for accessing a Dynatrace environment
type Environment struct {
	Name string     `yaml:"name"  json:"name" jsonschema:"required,description=The name of the environment - this can be freely defined and will be used in logs, etc."`
	URL  TypedValue `yaml:"url" json:"url" jsonschema:"required,oneof_type=string;object,description=The URL of the environment."`

	Auth Auth `yaml:"auth,omitempty" json:"auth" jsonschema:"required,description=This defines all information required for authenticated access to the environment's API."`
}

// Group defines a group of Environment
type Group struct {
	Name         string        `yaml:"name" json:"name" jsonschema:"required,description=The name of the group - this can be freely defined and will be used in logs, etc."`
	Environments []Environment `yaml:"environments" json:"environments" jsonschema:"required,minItems=1,description=The environments that are part of this group."`
}

type Manifest struct {
	ManifestVersion string `yaml:"manifestVersion" json:"manifestVersion"  jsonschema:"required,oneof_type=string;number,description=The version of this manifest. It is used when loading a manifest to ensure the CLI version is able to parse this manifest."`
	// Projects is a list of projects that will be deployed with this manifest
	Projects []Project `yaml:"projects" json:"projects" jsonschema:"minItems=1,description=A list of projects that will be deployed with this manifest"`
	// EnvironmentGroups is a list of environment groups that configs in Projects will be deployed to
	EnvironmentGroups []Group `yaml:"environmentGroups" json:"environmentGroups" jsonschema:"minItems=1,description=A list of environment groups that configs in the defined 'projects' will be deployed to. Required when deploying environment configurations."`
	// Accounts is a list of accounts that account resources in Projects will be deployed to
	Accounts []Account `yaml:"accounts,omitempty" json:"accounts" jsonschema:"minItems=1,description=A list of of accounts that account resources defined in 'projects' will be deployed to. Required when deploying account resources."`

	Parameters map[string]interface{} `yaml:"parameters,omitempty" json:"parameters"`
}

type Account struct {
	Name        string      `yaml:"name" json:"name" jsonschema:"description=The name of the account - this can be freely defined and will show up in logs, etc."`
	AccountUUID TypedValue  `yaml:"accountUUID" json:"accountUUID" jsonschema:"required,oneof_type=string;object,description=The uuid of your account - you can find this in the Account Management UI."`
	ApiUrl      *TypedValue `yaml:"apiUrl,omitempty" json:"apiUrl" jsonschema:"optional,oneof_type=string;object,default=api.dynatrace.com,description=Allows to optionally define a different Account Management API URL."`
	OAuth       OAuth       `yaml:"oAuth" json:"oAuth" jsonschema:"required,description=OAuth client credentials to authenticate API calls for this account."`
}
