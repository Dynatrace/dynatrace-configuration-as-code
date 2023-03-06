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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/regex"
	"strings"
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

// EnvironmentDefinition holds all information about a Dynatrace environment
type EnvironmentDefinition struct {
	Name  string
	Type  EnvironmentType
	url   UrlDefinition
	Group string
	Token Token
}

// UrlType describes from where the url is loaded.
// Possible values are [EnvironmentUrlType] and [ValueUrlType].
type UrlType string

const (
	// EnvironmentUrlType describes that the url has been loaded from an environment variable
	EnvironmentUrlType UrlType = "environment"

	// ValueUrlType describes that the url has been loaded directly as a value
	ValueUrlType = "value"
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

// Token is the API-Token for Dynatrace Platform API-Access
type Token struct {
	// Name is the name of the environment-variable of the token. It is used for converting monaco-v1 to monaco-v2 environments
	// where the value is not resolved, but the env-name has to be kept.
	Name string

	// Value holds the actual token value for the given [Name]. It is empty when converting vom monaco-v1 to monaco-v2
	Value string
}

type ProjectDefinitionByProjectId map[string]ProjectDefinition

// environmentV1 represents an environment as it was loaded for
// monaco v1
type environmentV1 interface {
	GetId() string
	GetEnvironmentUrl() string
	GetGroup() string
	GetTokenName() string
}

// NewEnvironmentDefinitionFromV1 creates an EnvironmentDefinition from an environment loaded by monaco v1
func NewEnvironmentDefinitionFromV1(env environmentV1, group string) EnvironmentDefinition {
	return EnvironmentDefinition{
		Name:  env.GetId(),
		url:   newUrlDefinitionFromV1(env),
		Group: group,
		Token: Token{Name: env.GetTokenName()},
	}
}

func newUrlDefinitionFromV1(env environmentV1) UrlDefinition {
	if regex.IsEnvVariable(env.GetEnvironmentUrl()) {
		// no need to resolve the value for conversion
		return UrlDefinition{
			Type: EnvironmentUrlType,
			Name: regex.TrimToEnvVariableName(env.GetEnvironmentUrl()),
		}
	}

	return UrlDefinition{
		Type:  ValueUrlType,
		Value: strings.TrimSuffix(env.GetEnvironmentUrl(), "/"),
	}
}

// NewEnvironmentDefinition creates a new EnvironmentDefinition
func NewEnvironmentDefinition(name string, url UrlDefinition, group string, token Token) EnvironmentDefinition {
	return EnvironmentDefinition{
		Name:  name,
		url:   url,
		Group: group,
		Token: token,
	}
}

func (e *EnvironmentDefinition) GetUrl() (string, error) {
	return e.url.Value, nil
}

// Environments is a map of environment-name -> EnvironmentDefinition
type Environments map[string]EnvironmentDefinition

type Manifest struct {
	// Projects defined in the manifest, split by project-name
	Projects ProjectDefinitionByProjectId

	// Environments defined in the manifest, split by environment-name
	Environments Environments
}

// FilterByNames filters the environments by name and returns all environments that match the given names.
// Given an empty slice, all environments are returned.
// The resulting slice is never empty.
//
// An error is returned if a given name is not available as environment
func (e Environments) FilterByNames(names []string) (Environments, error) {

	if len(names) == 0 {
		return e, nil
	}

	result := make(Environments, len(names))

	for _, environmentName := range names {
		if env, ok := e[environmentName]; ok {
			result[environmentName] = env
		} else {
			return nil, fmt.Errorf("environment '%s' not found", environmentName)
		}
	}

	return result, nil
}

// FilterByGroup returns all environments whose group-name matches the given name.
func (e Environments) FilterByGroup(groupName string) Environments {
	result := make(map[string]EnvironmentDefinition, len(e))

	for k, definition := range e {
		if definition.Group == groupName {
			result[k] = definition
		}
	}

	return result
}
