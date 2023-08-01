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

import (
	"reflect"
)

type GroupOverride struct {
	// Group name this override applies for
	Group string `yaml:"group" json:"group" jsonschema:"required"`
	// Override can contain any fields the base config definition does and overwrites their values for this envrionment
	Override ConfigDefinition `yaml:"override" json:"override" jsonschema:"required"`
}

type EnvironmentOverride struct {
	// Environment name this override applies for
	Environment string `yaml:"environment" json:"environment" jsonschema:"required"`
	// Override can contain any fields the base config definition does and overwrites their values for this envrionment
	Override ConfigDefinition `yaml:"override" json:"override" jsonschema:"required"`
}

type ConfigDefinition struct {
	// Name of this configuration - required for Classic Config API types
	Name ConfigParameter `yaml:"name,omitempty" json:"name,omitempty"`
	// Parameters for this configuration
	Parameters map[string]ConfigParameter `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	// Template defines the filepath to the JSON template used for this configuration
	Template string `yaml:"template,omitempty" json:"template,omitempty" jsonschema:"required"`
	// Skip defines whether this config should be skipped when deploying
	Skip ConfigParameter `yaml:"skip,omitempty" json:"skip,omitempty"`
	// OriginObjectId defines the ID of the Dynatrace object this config originated from - this is filled when downloading, but can also be set to tie a config to a specific object
	OriginObjectId string `yaml:"originObjectId,omitempty" json:"originObjectId,omitempty"`
}

type TopLevelConfigDefinition struct {
	// Id is the monaco identifier for this config - is used in references and for some generated IDs in Dynatrace environments
	Id string `yaml:"id" json:"id" jsonschema:"required"`
	// Config defines the actual configuration to be applied
	Config ConfigDefinition `yaml:"config" json:"config" jsonschema:"required"`
	// Type of this Config
	Type TypeDefinition `yaml:"type" json:"type" jsonschema:"required,oneof_type=string;object"`
	// GroupOverrides overwrite specific parts of the Config when deploying it to any environment in a given group
	GroupOverrides []GroupOverride `yaml:"groupOverrides,omitempty" json:"groupOverrides,omitempty"`
	// EnvironmentOverrides overwrite specific parts of the Config when deploying it to a given environment
	EnvironmentOverrides []EnvironmentOverride `yaml:"environmentOverrides,omitempty" json:"environmentOverrides,omitempty"`
}

type TopLevelDefinition struct {
	// Configs that will be applied to a Dynatrace environment
	Configs []TopLevelConfigDefinition `yaml:"configs" json:"configs" jsonschema:"required,minLength=1"`
}

type ConfigParameter interface{}

func GetTopLevelDefinitionYamlTypeName() string {
	return reflect.ValueOf(TopLevelDefinition{}).Type().String()
}
