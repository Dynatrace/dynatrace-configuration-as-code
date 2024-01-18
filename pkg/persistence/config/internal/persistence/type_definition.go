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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/mitchellh/mapstructure"
)

const BucketType = "bucket"

type TypeDefinition struct {
	Api        ApiDefinition        `yaml:"api,omitempty" json:"api,omitempty" jsonschema:"oneof_type=string;object"`
	Bucket     string               `yaml:"bucket,omitempty" json:"bucket,omitempty"`
	Settings   SettingsDefinition   `yaml:"settings,omitempty" json:"settings,omitempty"`
	Automation AutomationDefinition `yaml:"automation,omitempty" json:"automation,omitempty"`
}

type ApiDefinition any

func UnmarshalApiType(a ApiDefinition) (ComplexApiDefinition, error) {
	switch v := a.(type) {
	case string:
		return ComplexApiDefinition{
			Name: v,
		}, nil
	case map[any]any:
		var c ComplexApiDefinition
		if err := mapstructure.Decode(v, &c); err != nil {
			return ComplexApiDefinition{}, fmt.Errorf("failed to UnmarshalApiType api definition: %w", err)
		}

		return c, nil
	default:
		return ComplexApiDefinition{}, errors.New("unknown type in api definition")
	}
}

type ComplexApiDefinition struct {
	Name  string          `yaml:"name" json:"name" jsonschema:"required,description=The name of the API the config is for." mapstructure:"name"`
	Scope ConfigParameter `yaml:"scope,omitempty" json:"scope" jsonschema:"description=This defines the config where this config needs to be applied."  mapstructure:"scope"`
}

type SettingsDefinition struct {
	Schema        string          `yaml:"schema,omitempty" json:"schema,omitempty" jsonschema:"required,description=The Settings 2.0 schema of this config."`
	SchemaVersion string          `yaml:"schemaVersion,omitempty" json:"schemaVersion,omitempty" jsonschema:"description=This optionally informs the Settings API that a specific schema version was used for this config."`
	Scope         ConfigParameter `yaml:"scope,omitempty" json:"scope,omitempty"  jsonschema:"required,description=This defines the scope in which this Setting applies."`
}

type AutomationDefinition struct {
	Resource config.AutomationResource `yaml:"resource" json:"resource" jsonschema:"required,enum=workflow,enum=business-calendar,enum=scheduling-rule,description=This defines which automation resource this config is for."`
}

// UnmarshalYAML Custom unmarshaler that knows how to handle TypeDefinition.
// 'type' section can come as string or as struct as it is defind in `TypeDefinition`
// function parameter more than once if necessary.
func (c *TypeDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var data interface{}
	if err := unmarshal(&data); err != nil {
		return err
	}

	switch v := data.(type) {
	case string:
		if v == BucketType {
			// string was a bucket
			c.Bucket = v
			return nil
		}

		// string was a shorthand config API
		c.Api = v
		return nil
	default:
		var td TypeDefinition
		if err := mapstructure.Decode(v, &td); err == nil {
			*c = td
			return nil
		} else {
			return fmt.Errorf("failed to parse 'type' section: %w", err)
		}
	}
}

func (c *TypeDefinition) IsSound(knownApis map[string]struct{}) error {
	classicErrs := c.isClassicSound(knownApis)
	settingsErrs := c.Settings.isSettingsSound()
	automationErr := c.Automation.isSound()

	types := 0
	var err error

	if c.IsClassic() {
		types += 1
		err = classicErrs
	}
	if c.IsSettings() {
		types += 1
		err = settingsErrs
	}
	if c.IsAutomation() {
		types++
		err = automationErr
	}
	if c.IsBucket() {
		types++
	}

	typesSound := 0
	for _, e := range []error{classicErrs, settingsErrs, automationErr} {
		if e == nil {
			typesSound += 1
		}
	}

	switch {
	case types >= 2:
		return errors.New("wrong configuration of type property")
	case typesSound == 1:
		return nil
	case types == 0:
		return errors.New("type configuration is missing or unknown")
	case types == 1:
		return err
	default:
		return errors.New("wrong configuration of type property")
	}
}

// IsSettings returns true iff one of fields from TypeDefinition are filed up
func (c *TypeDefinition) IsSettings() bool {
	return c.Settings != SettingsDefinition{}
}
func (t *SettingsDefinition) isSettingsSound() error {
	var s []string
	if t.Schema == "" {
		s = append(s, "type.schema")
	}
	if t.Scope == nil {
		s = append(s, "type.scope")
	}
	if s == nil {
		return nil
	}
	return fmt.Errorf("next property missing: %v", s)
}

func (c *TypeDefinition) IsClassic() bool {
	a, err := UnmarshalApiType(c.Api)
	if err != nil {
		return false
	}

	return a.Name != ""
}
func (c *TypeDefinition) isClassicSound(knownApis map[string]struct{}) error {
	a, err := UnmarshalApiType(c.Api)
	if err != nil {
		return err
	}

	if _, found := knownApis[a.Name]; !found {
		return errors.New("unknown API: " + a.Name)
	}
	return nil
}

func (c *TypeDefinition) IsAutomation() bool {
	return c.Automation != AutomationDefinition{}
}

func (c *AutomationDefinition) isSound() error {

	switch c.Resource {
	case "":
		return errors.New("missing 'type.automation.resource' property")

	case config.Workflow, config.BusinessCalendar, config.SchedulingRule:
		return nil

	default:
		return fmt.Errorf("unknown automation resource %q", c.Resource)
	}
}

func (c *TypeDefinition) IsBucket() bool {
	return c.Bucket != ""
}

func (c *TypeDefinition) GetApiType() (string, error) {
	switch {
	case c.IsSettings():
		return c.Settings.Schema, nil
	case c.IsClassic():
		a, err := UnmarshalApiType(c.Api)
		if err != nil {
			return "", err
		}
		return a.Name, nil

	case c.IsAutomation():
		return string(c.Automation.Resource), nil
	case c.IsBucket():
		return BucketType, nil
	default:
		return "", errors.New("missing type definition")
	}
}
