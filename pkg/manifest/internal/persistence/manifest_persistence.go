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
	Name string `yaml:"name" json:"name"`
	Type string `yaml:"type,omitempty" json:"type"`
	Path string `yaml:"path,omitempty" json:"path"`
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
	Type  Type   `yaml:"type,omitempty" mapstructure:"type" json:"type"`
	Value string `yaml:"value" mapstructure:"value" json:"value"`
}

// UnmarshalYAML Custom unmarshaler for TypedValue able to parse simple shorthands (accountUUID: 1234) and full values.
func (c *TypedValue) UnmarshalYAML(unmarshal func(any) error) error {
	var data any
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch data := data.(type) {
	case string:
		c.Type = TypeValue
		c.Value = data
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
	Name string `yaml:"name" json:"name"`
}

// OAuth defines the required information to request oAuth bearer tokens for authenticated API calls
type OAuth struct {
	// ClientID of the oAuth client credentials used to request bearer-tokens for authenticated API calls
	ClientID AuthSecret `yaml:"clientId" json:"clientId"`
	// ClientSecret of the oAuth client credentials used to request bearer-tokens for authenticated API calls
	ClientSecret AuthSecret `yaml:"clientSecret" json:"clientSecret"`
	// TokenEndpoint allows to optionally define a non-standard endpoint to request bearer-tokens from. Defaults to production sso.dynatrace.com if not defined.
	TokenEndpoint *TypedValue `yaml:"tokenEndpoint,omitempty" json:"tokenEndpoint"`
}

// Auth defines all required information for authenticated API calls
type Auth struct {
	// AccessToken defines an API access tokens used for Dynatrace Config API calls
	AccessToken *AuthSecret `yaml:"token,omitempty" json:"token"`
	// OAuth defines client credentials used for Dynatrace Platform API calls
	OAuth *OAuth `yaml:"oAuth,omitempty" json:"oAuth"`
	// PlatformToken defines a platform token used for Dynatrace Platform API calls
	PlatformToken *AuthSecret `yaml:"platformToken,omitempty" json:"platformToken"`
}

// Environment defines all required information for accessing a Dynatrace environment
type Environment struct {
	Name string     `yaml:"name"  json:"name"`
	URL  TypedValue `yaml:"url" json:"url"`

	Auth Auth `yaml:"auth,omitempty" json:"auth"`
}

// Group defines a group of Environment
type Group struct {
	Name         string        `yaml:"name" json:"name"`
	Environments []Environment `yaml:"environments" json:"environments"`
}

type Manifest struct {
	ManifestVersion string `yaml:"manifestVersion" json:"manifestVersion" `
	// Projects is a list of projects that will be deployed with this manifest
	Projects []Project `yaml:"projects" json:"projects"`
	// EnvironmentGroups is a list of environment groups that configs in Projects will be deployed to
	EnvironmentGroups []Group `yaml:"environmentGroups" json:"environmentGroups"`
	// Accounts is a list of accounts that account resources in Projects will be deployed to
	Accounts []Account `yaml:"accounts,omitempty" json:"accounts"`
}

type Account struct {
	Name        string      `yaml:"name" json:"name"`
	AccountUUID TypedValue  `yaml:"accountUUID" json:"accountUUID"`
	ApiUrl      *TypedValue `yaml:"apiUrl,omitempty" json:"apiUrl"`
	OAuth       OAuth       `yaml:"oAuth" json:"oAuth"`
}
