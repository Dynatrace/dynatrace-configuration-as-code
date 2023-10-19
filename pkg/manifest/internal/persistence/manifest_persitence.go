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

const SimpleProjectType = "simple"
const GroupProjectType = "grouping"

type Project struct {
	Name string `yaml:"name"`
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty"`
}

type SecretType string

const TypeEnvironment SecretType = "environment"

// AuthSecret represents a user-defined client id or client secret. It has a [Type] which is [TypeEnvironment] (default).
// Secrets must never be provided as plain text, but always loaded from somewhere else. Currently, loading is only allowed from environment variables.
//
// [Name] contains the environment-variable to resolve the authSecret.
//
// This struct is meant to be reused for fields that require the same behavior.
type AuthSecret struct {
	Type SecretType `yaml:"type"`
	Name string     `yaml:"name"`
}

type OAuth struct {
	ClientID      AuthSecret `yaml:"clientId"`
	ClientSecret  AuthSecret `yaml:"clientSecret"`
	TokenEndpoint *Url       `yaml:"tokenEndpoint,omitempty"`
}

type Auth struct {
	Token AuthSecret `yaml:"token"`
	OAuth *OAuth     `yaml:"oAuth,omitempty"`
}

type Environment struct {
	Name string `yaml:"name"`
	URL  Url    `yaml:"url"`

	// Auth contains all authentication related information
	Auth Auth `yaml:"auth,omitempty"`
}

type UrlType string

const (
	UrlTypeEnvironment UrlType = "environment"
	UrlTypeValue       UrlType = "value"
)

type Url struct {
	Type  UrlType `yaml:"type,omitempty"`
	Value string  `yaml:"value"`
}

type Group struct {
	Name         string        `yaml:"name"`
	Environments []Environment `yaml:"environments"`
}

type Manifest struct {
	ManifestVersion   string    `yaml:"manifestVersion"`
	Projects          []Project `yaml:"projects"`
	EnvironmentGroups []Group   `yaml:"environmentGroups"`
	Accounts          []Account `yaml:"accounts,omitempty"`
}

type Account struct {
	Name        string `yaml:"name"`
	AccountUUID string `yaml:"accountUUID"`
	ApiUrl      *Url   `yaml:"apiUrl,omitempty"`
	OAuth       OAuth  `yaml:"oAuth"`
}
