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

package v2

import (
	"fmt"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
)

// ConfigsPerType is a map of configType (api or schema id) to configs
type ConfigsPerType map[string][]config.Config

type EntitiesPerType map[string][]string

// ConfigsPerTypePerEnvironments is a map of environment to api to configs
type ConfigsPerTypePerEnvironments map[string]ConfigsPerType

// ConfigsPerEnvironment is a map of environment to configs
type ConfigsPerEnvironment map[string][]config.Config

// DependenciesPerEnvironment is a map of environment to project ids
type DependenciesPerEnvironment map[string][]string

type Project struct {
	Id string

	// set to the name defined in manifest if this project is part of a grouping, else will be empty
	GroupId string

	Configs ConfigsPerTypePerEnvironments

	// map of environment to project ids
	Dependencies DependenciesPerEnvironment
}

func (p Project) HasDependencyOn(environment string, project Project) bool {
	dependencies, found := p.Dependencies[environment]

	if !found {
		return false
	}

	for _, dep := range dependencies {
		if dep == project.Id {
			return true
		}
	}

	return false
}

func (p Project) String() string {
	if p.GroupId != "" {
		return fmt.Sprintf("%s [group: %s]", p.Id, p.GroupId)
	}

	return p.Id
}
