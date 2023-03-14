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

package manifest

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
)

// EnvironmentType is used to identify the type of the environment.
// Possible values are  [Classic] and [Platform]
type EnvironmentType int

const (
	// Classic identifies a Dynatrace Classic environment
	Classic EnvironmentType = iota

	// Platform identifies a Dynatrace Platform environment
	Platform
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
	ClientId     AuthSecret
	ClientSecret AuthSecret
}

type Auth struct {
	Token AuthSecret
	OAuth OAuth
}

// EnvironmentDefinition holds all information about a Dynatrace environment
type EnvironmentDefinition struct {
	Name  string
	Type  EnvironmentType
	Url   UrlDefinition
	Group string

	Auth Auth
}

// UrlType describes from where the url is loaded.
// Possible values are [EnvironmentUrlType] and [ValueUrlType].
// [ValueUrlType] is the default value.
type UrlType int

const (
	// ValueUrlType describes that the url has been loaded directly as a value
	ValueUrlType UrlType = iota

	// EnvironmentUrlType describes that the url has been loaded from an environment variable
	EnvironmentUrlType
)

// UrlDefinition holds the value and origin of an environment-url.
type UrlDefinition struct {
	// Type defines whether the [UrlDefinition.Value] is loaded from an env var, or directly.
	Type UrlType

	// Name is the name of the environment-variable of the token. It only has a value if [UrlDefinition.Type] is "[EnvironmentUrlType]"
	Name string

	// Value is the resolved value of the Url.
	// It is resolved during manifest reading.
	Value string
}

// AuthSecret contains a resolved secret value. It is used for the API-Token, ClientId, and ClientSecret.
type AuthSecret struct {
	// Name is the name of the environment-variable of the token. It is used for converting monaco-v1 to monaco-v2 environments
	// where the value is not resolved, but the env-name has to be kept.
	Name string

	// Value holds the actual token value for the given [Name]. It is empty when converting vom monaco-v1 to monaco-v2
	Value string
}

type ProjectDefinitionByProjectId map[string]ProjectDefinition

// Environments is a map of environment-name -> EnvironmentDefinition
type Environments map[string]EnvironmentDefinition

// Names returns the slice of environment names
func (e Environments) Names() []string {
	return maps.Keys(e)
}

type Manifest struct {
	// Projects defined in the manifest, split by project-name
	Projects ProjectDefinitionByProjectId

	// Environments defined in the manifest, split by environment-name
	Environments Environments
}
