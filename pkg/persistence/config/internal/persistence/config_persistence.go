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
	Group    string           `yaml:"group" json:"group" jsonschema:"required,description=Name of the group this override applies for."`
	Override ConfigDefinition `yaml:"override" json:"override" jsonschema:"required,description=May contain any fields the base config definition does and overwrites their values for any environment in this group."`
}

type EnvironmentOverride struct {
	Environment string           `yaml:"environment" json:"environment" jsonschema:"required,description=Name of the environment this override applies for."`
	Override    ConfigDefinition `yaml:"override" json:"override" jsonschema:"required,description=May contain any fields the base config definition does and overwrites their values for this environment."`
}

type ConfigDefinition struct {
	Name           ConfigParameter            `yaml:"name,omitempty" json:"name,omitempty" jsonschema:"description=The name of this configuration - required for Classic Config API types."`
	Parameters     map[string]ConfigParameter `yaml:"parameters,omitempty" json:"parameters,omitempty" jsonschema:"description=Parameters for this configuration."`
	Template       string                     `yaml:"template,omitempty" json:"template,omitempty" jsonschema:"required,description=The filepath to the JSON template used for this configuration"`
	Skip           ConfigParameter            `yaml:"skip,omitempty" json:"skip,omitempty" jsonschema:"description=Defines whether this config should be skipped when deploying."`
	OriginObjectId string                     `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty" jsonschema:"description=description=The identifier of the Dynatrace object this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object."`
}

type TopLevelConfigDefinition struct {
	Id     string           `yaml:"id" json:"id" jsonschema:"required,description=The monaco identifier for this config - is used in references and for some generated IDs in Dynatrace environments."`
	Config ConfigDefinition `yaml:"config" json:"config" jsonschema:"required,description=The actual configuration to be applied"`
	Type   TypeDefinition   `yaml:"type" json:"type" jsonschema:"required,oneof_type=string;object,description=The type of this configuration, e.g. a config API or a Settings 2.0 schema."`
	// GroupOverrides overwrite specific parts of the Config when deploying it to any environment in a given group
	GroupOverrides []GroupOverride `yaml:"groupOverrides,omitempty" json:"groupOverrides,omitempty" jsonschema:"description=GroupOverrides overwrite specific parts of the Config when deploying it to any environment in a given group."`
	// EnvironmentOverrides overwrite specific parts of the Config when deploying it to a given environment
	EnvironmentOverrides []EnvironmentOverride `yaml:"environmentOverrides,omitempty" json:"environmentOverrides,omitempty" jsonschema:"description=EnvironmentOverrides overwrite specific parts of the Config when deploying it to a given environment."`
}

type TopLevelDefinition struct {
	Configs []TopLevelConfigDefinition `yaml:"configs" json:"configs" jsonschema:"required,minItems=1,description=The configurations that will be applied to a Dynatrace environment."`
}

type ConfigParameter interface{}
