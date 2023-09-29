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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/mitchellh/mapstructure"
)

const BucketType = "bucket"

type TypeDefinition struct {
	Api        string               `yaml:"api,omitempty"`
	Bucket     string               `yaml:"bucket,omitempty"`
	Settings   SettingsDefinition   `yaml:"settings,omitempty"`
	Automation AutomationDefinition `yaml:"automation,omitempty"`
}

type SettingsDefinition struct {
	Schema        string          `yaml:"schema,omitempty"`
	SchemaVersion string          `yaml:"schemaVersion,omitempty"`
	Scope         ConfigParameter `yaml:"scope,omitempty"`
}

type AutomationDefinition struct {
	Resource config.AutomationResource `yaml:"resource"`
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
			if !featureflags.Buckets().Enabled() {
				return fmt.Errorf("failed to parse 'type' section: unknown type %q", v)
			}

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
	return c.Api != ""
}
func (c *TypeDefinition) isClassicSound(knownApis map[string]struct{}) error {
	if !c.IsClassic() {
		return errors.New("missing 'type.api' property")
	}

	if _, found := knownApis[c.Api]; !found {
		return errors.New("unknown API: " + c.Api)
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

func (c *TypeDefinition) GetApiType() string {
	switch {
	case c.IsSettings():
		return c.Settings.Schema
	case c.IsClassic():
		return c.Api
	case c.IsAutomation():
		return string(c.Automation.Resource)
	case c.IsBucket():
		return BucketType
	default:
		return ""
	}
}
