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
	environmentv1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/envvars"
	"strings"
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
		Name: env.GetId(),
		url: UrlDefinition{
			Type:  ValueUrlType,
			Value: strings.TrimSuffix(env.GetEnvironmentUrl(), "/"),
		},
		Group: group,
		Token: &EnvironmentVariableToken{EnvironmentVariableName: env.GetTokenName()},
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
	if token, found := envvars.Lookup(t.EnvironmentVariableName); found {
		return token, nil
	}

	return "", fmt.Errorf("no environment variable `%s` set", t.EnvironmentVariableName)
}

func (e *EnvironmentDefinition) GetUrl() (string, error) {

	switch e.url.Type {
	case EnvironmentUrlType:
		if url, found := envvars.Lookup(e.url.Value); found {
			return url, nil
		} else {
			return "", fmt.Errorf("no environment variable set for %s", e.url.Value)
		}
	case ValueUrlType:
		return e.url.Value, nil
	default:
		return "", fmt.Errorf("type `%s` does not exist for enviroment URL. Supported are %s and %s", e.url.Type, EnvironmentUrlType, ValueUrlType)
	}
}

type Manifest struct {
	Projects     map[string]ProjectDefinition
	Environments map[string]EnvironmentDefinition
}

func (m *Manifest) GetEnvironmentsAsSlice() []EnvironmentDefinition {
	result := make([]EnvironmentDefinition, 0, len(m.Environments))

	for _, env := range m.Environments {
		result = append(result, env)
	}

	return result
}
