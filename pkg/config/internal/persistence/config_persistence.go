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

type GroupOverride struct {
	Group    string           `yaml:"group" json:"group"`
	Override ConfigDefinition `yaml:"override" json:"override"`
}

type EnvironmentOverride struct {
	Environment string           `yaml:"environment" json:"environment"`
	Override    ConfigDefinition `yaml:"override" json:"override"`
}

type ConfigDefinition struct {
	Name           ConfigParameter            `yaml:"name,omitempty" json:"name,omitempty"`
	Parameters     map[string]ConfigParameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Template       string                     `yaml:"template,omitempty" json:"template,omitempty"`
	Skip           ConfigParameter            `yaml:"skip,omitempty" json:"skip,omitempty"`
	OriginObjectId string                     `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
}

type TopLevelConfigDefinition struct {
	Id     string           `yaml:"id" json:"id"`
	Config ConfigDefinition `yaml:"config" json:"config"`
	Type   TypeDefinition   `yaml:"type" json:"type"`
	// GroupOverrides overwrite specific parts of the Config when deploying it to any environment in a given group
	GroupOverrides []GroupOverride `yaml:"groupOverrides,omitempty" json:"groupOverrides,omitempty"`
	// EnvironmentOverrides overwrite specific parts of the Config when deploying it to a given environment
	EnvironmentOverrides []EnvironmentOverride `yaml:"environmentOverrides,omitempty" json:"environmentOverrides,omitempty"`
}

type TopLevelDefinition struct {
	Configs []TopLevelConfigDefinition `yaml:"configs" json:"configs"`
}

type ConfigParameter any
