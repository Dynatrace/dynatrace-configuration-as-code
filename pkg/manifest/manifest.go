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
	"os"
)

type ProjectDefinition struct {
	Name string
	Path string
}

type EnvironmentDefinition struct {
	Name  string
	Url   string
	Group string
	Token
}
type Token interface {
	GetToken() (string, error)
}

type EnvironmentVariableToken struct {
	EnvironmentVariableName string
}

func (t *EnvironmentVariableToken) GetToken() (string, error) {
	if token, found := os.LookupEnv(t.EnvironmentVariableName); found {
		return token, nil
	}

	return "", fmt.Errorf("no environment variable `%s` set", t.EnvironmentVariableName)
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
