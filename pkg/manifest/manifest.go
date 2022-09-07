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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"os"
	"strings"

	environmentv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
)

type ProjectDefinition struct {
	Name string
	Path string
}

type EnvironmentDefinition struct {
	Name  string
	url   UrlDefinition
	Group string
	Token
}

type UrlType string

const EnvironmentUrlType UrlType = "environment"
const ValueUrlType UrlType = "value"

type UrlDefinition struct {
	Type  UrlType
	Value string
}

type Token interface {
	GetToken() (string, error)
}

type EnvironmentVariableToken struct {
	EnvironmentVariableName string
}

func NewEnvironmentDefinitionFromV1(env environmentv1.Environment, group string) EnvironmentDefinition {
	return EnvironmentDefinition{
		Name:  env.GetId(),
		url:   newUrlDefinitionFromV1(env),
		Group: group,
		Token: &EnvironmentVariableToken{EnvironmentVariableName: env.GetTokenName()},
	}
}

func newUrlDefinitionFromV1(env environmentv1.Environment) UrlDefinition {
	if util.IsEnvVariable(env.GetEnvironmentUrl()) {
		return UrlDefinition{
			Type:  EnvironmentUrlType,
			Value: util.TrimToEnvVariableName(env.GetEnvironmentUrl()),
		}
	}

	return UrlDefinition{
		Type:  ValueUrlType,
		Value: strings.TrimSuffix(env.GetEnvironmentUrl(), "/"),
	}
}

func NewEnvironmentDefinition(name string, url UrlDefinition, group string, token *EnvironmentVariableToken) EnvironmentDefinition {
	return EnvironmentDefinition{
		Name:  name,
		url:   url,
		Group: group,
		Token: token,
	}
}

func (t *EnvironmentVariableToken) GetToken() (string, error) {
	if token, found := os.LookupEnv(t.EnvironmentVariableName); found {
		return token, nil
	}

	return "", fmt.Errorf("no environment variable `%s` set", t.EnvironmentVariableName)
}

func (e *EnvironmentDefinition) GetUrl() (string, error) {

	switch e.url.Type {
	case EnvironmentUrlType:
		if url, found := os.LookupEnv(e.url.Value); found {
			return url, nil
		} else {
			return "", fmt.Errorf("no environment variable set for %s", e.url.Value)
		}
	case ValueUrlType:
		return e.url.Value, nil
	default:
		return "", fmt.Errorf("url.type `%s` is not a valid type for enviroment URL. Supported are %s and %s", e.url.Type, EnvironmentUrlType, ValueUrlType)
	}
}

type Manifest struct {
	// Projects defined in the manifest, split by project-name
	Projects map[string]ProjectDefinition

	// Environments defined in the manifest, split by environment-name
	Environments map[string]EnvironmentDefinition
}

func (m *Manifest) GetEnvironmentsAsSlice() []EnvironmentDefinition {
	result := make([]EnvironmentDefinition, 0, len(m.Environments))

	for _, env := range m.Environments {
		result = append(result, env)
	}

	return result
}

// FilterEnvironmentsByNames filters the environments by name and returns all environments that match the given names.
// Given an empty slice, all environments are returned.
// The resulting slice is never empty.
//
// An error is returned if a given name is not available as environment
func (m *Manifest) FilterEnvironmentsByNames(names []string) ([]EnvironmentDefinition, error) {

	if len(names) == 0 {
		return m.GetEnvironmentsAsSlice(), nil
	}

	result := make([]EnvironmentDefinition, 0, len(names))

	for _, environmentName := range names {
		if env, ok := m.Environments[environmentName]; ok {
			result = append(result, env)
		} else {
			return nil, fmt.Errorf("environment '%s' not found", environmentName)
		}
	}

	return result, nil
}
