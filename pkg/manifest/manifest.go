// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package manifest and its subpackages contains all information regarding the in-memory definitions of manifests, as well as the
// persistence layer, loading, and writing.
// To load use the [loader] package, to write use the [writer] package.
package manifest

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/oauth2/endpoints"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

type ProjectDefinition struct {
	Name  string
	Group string
	Path  string
}

func (p ProjectDefinition) String() string {
	if p.Group != "" {
		return fmt.Sprintf("%s (group: %s, path: %s)", p.Name, p.Group, p.Path)
	}
	return fmt.Sprintf("%s (path: %s)", p.Name, p.Path)
}

type OAuth struct {
	ClientID      AuthSecret
	ClientSecret  AuthSecret
	TokenEndpoint *URLDefinition
}

// GetTokenEndpointValue returns the defined token endpoint or the default token endpoint if none is defined
func (o OAuth) GetTokenEndpointValue() string {
	if o.TokenEndpoint == nil || o.TokenEndpoint.Value == "" {
		return endpoints.Dynatrace.TokenURL
	}
	return o.TokenEndpoint.Value
}

type Auth struct {
	Token *AuthSecret
	OAuth *OAuth
}

// EnvironmentDefinition holds all information about a Dynatrace environment
type EnvironmentDefinition struct {
	Name  string
	Group string
	URL   URLDefinition
	Auth  Auth
}

// URLType describes from where the url is loaded.
// Possible values are [EnvironmentURLType] and [ValueURLType].
// [ValueURLType] is the default value.
type URLType int

const (
	// ValueURLType describes that the url has been loaded directly as a value
	ValueURLType URLType = iota

	// EnvironmentURLType describes that the url has been loaded from an environment variable
	EnvironmentURLType
)

// URLDefinition holds the value and origin of an environment-url.
type URLDefinition struct {
	// Type defines whether the [URLDefinition.Value] is loaded from an env var, or directly.
	Type URLType

	// Name is the name of the environment-variable of the token. It only has a value if [URLDefinition.Type] is "[EnvironmentURLType]"
	Name string

	// Value is the resolved value of the URL.
	// It is resolved during manifest reading.
	Value string
}

// AuthSecret contains a resolved secret value. It is used for the API-Token, ClientID, and ClientSecret.
type AuthSecret struct {
	// Name is the name of the environment-variable of the token. It is used for converting monaco-v1 to monaco-v2 environments
	// where the value is not resolved, but the env-name has to be kept.
	Name string

	// Value holds the actual token value for the given [Name]. It is empty when converting vom monaco-v1 to monaco-v2
	Value secret.MaskedString
}

type ProjectDefinitionByProjectID map[string]ProjectDefinition

// Environments is a map of environment-name -> EnvironmentDefinition
type Environments map[string]EnvironmentDefinition

// Names returns the slice of environment names
func (e Environments) Names() []string {
	return maps.Keys(e)
}

// Account holds all necessary information to access the account API
type Account struct {
	// Name is the account-name that is used to resolve user-defined lookup names.
	Name string

	// AccountUUID is the Dynatrace-account UUID
	AccountUUID uuid.UUID

	// ApiUrl is the target URL of this account.
	// It is used when the default account management url is not the target account management url.
	ApiUrl *URLDefinition

	// OAuth holds the OAuth credentials used to access the account API.
	OAuth OAuth
}

// Manifest is the central component. It holds all information that is needed to deploy projects.
type Manifest struct {
	// Projects defined in the manifest, split by project-name
	Projects ProjectDefinitionByProjectID

	// Environments defined in the manifest, split by environment-name
	Environments Environments

	// Accounts holds all accounts defined in the manifest. Key is the user-defined account name.
	Accounts map[string]Account

	Parameters map[string]parameter.Parameter
}
