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

type groupOverride struct {
	Group    string           `yaml:"group"`
	Override configDefinition `yaml:"override"`
}

type environmentOverride struct {
	Environment string           `yaml:"environment"`
	Override    configDefinition `yaml:"override"`
}

type configDefinition struct {
	Name       configParameter            `yaml:"name,omitempty"`
	Parameters map[string]configParameter `yaml:"parameters,omitempty"`
	Template   string                     `yaml:"template,omitempty"`
	Skip       interface{}                `yaml:"skip,omitempty"`
}

type topLevelConfigDefinition struct {
	Id                   string                `yaml:"id"`
	Config               configDefinition      `yaml:"config"`
	GroupOverrides       []groupOverride       `yaml:"groupOverrides,omitempty"`
	EnvironmentOverrides []environmentOverride `yaml:"environmentOverrides,omitempty"`
}

type topLevelDefinition struct {
	Configs []topLevelConfigDefinition
}

type configParameter interface{}

type newTopLevelDefinition struct {
	Version newConfigDefinitionVersion
	Configs []newTopLevelConfigDefinition
}

type newConfigDefinitionVersion string
const v2 newConfigDefinitionVersion = "v2"

type newTopLevelConfigDefinition struct {
	Config               newConfigDefinition      				 `yaml:"config"`
	GroupOverrides       []groupOverride                         `yaml:"groupOverrides,omitempty"`
	EnvironmentOverrides []environmentOverride                   `yaml:"environmentOverrides,omitempty"`
}

type newConfigDefinition struct {

	// Either Name or NameFromReference MUST be set
	// If none is set -> Error
	// If both are set -> Error
	Name                 string                                  `yaml:"name,omitempty"`
	NameFromReference    newConfigParameterDefinition            `yaml:"nameFromReference,omitempty"`

	Parameters           map[string]newConfigParameterDefinition `yaml:"parameters,omitempty"`
	Template             string                                  `yaml:"template,omitempty"`
	Skip                 bool                                    `yaml:"skip,omitempty"`
}

type newConfigParameterDefinition struct {
	Type     ConfigParameterType `yaml:"type,omitempty"`
	Property string              `yaml:"property,omitempty"`
	ConfigId string              `yaml:"config-id,omitempty"`
	Api      string              `yaml:"api,omitempty"`
}

type ConfigParameterType string
const ConfigParameterReference ConfigParameterType = "reference"

