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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

type (
	EnvironmentName = string
	// ConfigsPerTypePerEnvironments is a map of EnvironmentName to a ConfigsPerType map
	ConfigsPerTypePerEnvironments map[EnvironmentName]ConfigsPerType

	ConfigTypeName = string
	// ConfigsPerType is a map of ConfigTypeName string (e.g. API ID, settings schema, automation resource, ...) to configs of that type
	ConfigsPerType map[ConfigTypeName][]config.Config

	// ConfigsPerEnvironment is a map of EnvironmentName to configs. This is a flattened version of ConfigsPerTypePerEnvironments
	ConfigsPerEnvironment map[EnvironmentName][]config.Config

	ProjectID = string
	// DependenciesPerEnvironment is a map of EnvironmentName to project IDs
	DependenciesPerEnvironment map[EnvironmentName][]ProjectID

	// ActionOverConfig is a function that will be performed over each config that is part of a project via a Project.ForEveryConfigDo method
	ActionOverConfig func(c config.Config)
)

type Project struct {
	Id string

	// set to the name defined in manifest if this project is part of a grouping, else will be empty
	GroupId string

	// Configs are the configurations within this Project
	Configs ConfigsPerTypePerEnvironments

	// Dependencies of this project to other projects
	Dependencies DependenciesPerEnvironment
}

// HasDependencyOn returns whether the project it is called on, has a dependency on the given project, for the given environment
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

// GetConfigFor searches a config object for matching the given coordinate in the
// current project.
func (p Project) GetConfigFor(env string, c coordinate.Coordinate) (config.Config, bool) {
	configsPerEnvironments, f := p.Configs[env]
	if !f {
		return config.Config{}, false
	}

	for cType, configsPerType := range configsPerEnvironments {
		if c.Type == cType {
			for _, cfg := range configsPerType {
				if cfg.Coordinate.ConfigId == c.ConfigId {
					return cfg, true
				}
			}
		}
	}

	return config.Config{}, false
}

func (p Project) String() string {
	if p.GroupId != "" {
		return fmt.Sprintf("%s [group: %s]", p.Id, p.GroupId)
	}

	return p.Id
}

// ForEveryConfigDo executes the given ActionOverConfig actions for each configuration defined in the project for each environment
// Actions can not modify the configs inside the Project.
func (p Project) ForEveryConfigDo(actions ...ActionOverConfig) {
	p.forEveryConfigDo("", actions)
}

// ForEveryConfigInEnvironmentDo executes the given ActionOverConfig actions for each configuration defined in the project for a given environment.
// It behaves like ForEveryConfigDo just limited to a single environment.
// Actions can not modify the configs inside the Project.
func (p Project) ForEveryConfigInEnvironmentDo(environment string, actions ...ActionOverConfig) {
	p.forEveryConfigDo(environment, actions)
}

// forEveryConfigDo applies the given action to every configuration, either for a single environment if requested,
// or for all environments if the environemnt parameter is empty.
func (p Project) forEveryConfigDo(environment string, actions []ActionOverConfig) {
	for env, cpt := range p.Configs {
		if environment == "" || environment == env {
			cpt.ForEveryConfigDo(actions...)
		}
	}
}

// ForEveryConfigDo executes the given ActionOverConfig actions for each configuration defined in the ConfigsPerType.
// Actions can not modify the configs inside the ConfigsPerType.
func (cpt ConfigsPerType) ForEveryConfigDo(actions ...ActionOverConfig) {
	for _, cs := range cpt {
		for _, c := range cs {
			for _, f := range actions {
				f(c)
			}
		}
	}
}

// AllConfigs is an iterator iterating over all configs
func (cpt ConfigsPerType) AllConfigs(yield func(config.Config) bool) {
	for _, cs := range cpt {
		for _, c := range cs {
			if !yield(c) {
				return
			}
		}
	}
}
