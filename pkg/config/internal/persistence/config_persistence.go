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
	Group    string           `yaml:"group"`
	Override ConfigDefinition `yaml:"override"`
}

type EnvironmentOverride struct {
	Environment string           `yaml:"environment"`
	Override    ConfigDefinition `yaml:"override"`
}

type ConfigDefinition struct {
	Name           ConfigParameter            `yaml:"name,omitempty"`
	Parameters     map[string]ConfigParameter `yaml:"parameters,omitempty"`
	Template       string                     `yaml:"template,omitempty"`
	Skip           ConfigParameter            `yaml:"skip,omitempty"`
	OriginObjectId string                     `yaml:"originObjectId,omitempty"`
}

type TopLevelConfigDefinition struct {
	Id                   string                `yaml:"id"`
	Config               ConfigDefinition      `yaml:"config"`
	Type                 TypeDefinition        `yaml:"type"`
	GroupOverrides       []GroupOverride       `yaml:"groupOverrides,omitempty"`
	EnvironmentOverrides []EnvironmentOverride `yaml:"environmentOverrides,omitempty"`
}

type TopLevelDefinition struct {
	Configs []TopLevelConfigDefinition `yaml:"configs"`
}

type ConfigParameter interface{}

func GetTopLevelDefinitionYamlTypeName() string {
	return reflect.ValueOf(TopLevelDefinition{}).Type().String()
}
