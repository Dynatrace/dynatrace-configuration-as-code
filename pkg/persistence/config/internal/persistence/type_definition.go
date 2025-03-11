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
	"slices"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

const (
	BucketType                = "bucket"
	SegmentType               = "segment"
	ServiceLevelObjectiveType = "slo-v2"
)

type TypeDefinition struct {
	Type        config.Type
	Scope       ConfigParameter
	InsertAfter ConfigParameter
}

type ComplexApiDefinition struct {
	Name  string          `yaml:"name" json:"name" jsonschema:"required,description=The name of the API the config is for." mapstructure:"name"`
	Scope ConfigParameter `yaml:"scope,omitempty" json:"scope" jsonschema:"description=This defines the config where this config needs to be applied."  mapstructure:"scope"`
}

type SettingsDefinition struct {
	Schema        string                `yaml:"schema,omitempty" json:"schema,omitempty" jsonschema:"required,description=The Settings 2.0 schema of this config."`
	SchemaVersion string                `yaml:"schemaVersion,omitempty" json:"schemaVersion,omitempty" jsonschema:"description=This optionally informs the Settings API that a specific schema version was used for this config."`
	Scope         ConfigParameter       `yaml:"scope,omitempty" json:"scope,omitempty"  jsonschema:"required,description=This defines the scope in which this Setting applies."`
	InsertAfter   ConfigParameter       `yaml:"insertAfter,omitempty" json:"insertAfter,omitempty" jsonschema:"description=This optionally informs the settings API that this particular objects needs to be inserted after the referenced one."`
	Permissions   *PermissionDefinition `yaml:"permissions,omitempty" json:"permissions,omitempty" jsonschema:"description=Some description"`
}

type PermissionDefinition struct {
	AllUsers *string `yaml:"all-users,omitempty" json:"all-users" jsonschema:"required,enum=read,enum=write,enum=none,description=All users can use this permission." mapstructure:"all-users"`
}

type AutomationDefinition struct {
	Resource config.AutomationResource `yaml:"resource" json:"resource" jsonschema:"required,enum=workflow,enum=business-calendar,enum=scheduling-rule,description=This defines which automation resource this config is for."`
}

type DocumentDefinition struct {
	Kind    config.DocumentKind `yaml:"kind" json:"kind" jsonschema:"required,enum=dashboard,enum=notebook,description=This defines the kind of document this config is for." mapstructure:"kind"`
	Private bool                `yaml:"private,omitempty" json:"private,omitempty" jsonschema:"description=Set to true to make the document private"  mapstructure:"private"`
}

type OpenPipelineDefinition struct {
	Kind string `yaml:"kind" json:"kind" jsonschema:"required,description=This defines the kind of OpenPipeline this config is for." mapstructure:"kind"`
}

// UnmarshalYAML Custom unmarshaler that knows how to handle TypeDefinition.
// 'type' section can come as string or as struct as it is defind in `TypeDefinition`
// function parameter more than once if necessary.
func (c *TypeDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {

	// The TypeDefinition allows for the shorthand syntax of `api: my-api`.
	// To catch that, let's try to unmarshal directly into a string. If it works, we know the shorthand is used.
	var str string
	if err := unmarshal(&str); err == nil {
		switch str {
		case BucketType:
			c.Type = config.BucketType{}
		case SegmentType:
			if !featureflags.Segments.Enabled() {
				return fmt.Errorf("unknown config-type %q", str)
			}
			c.Type = config.Segment{}
		case ServiceLevelObjectiveType:
			if !featureflags.ServiceLevelObjective.Enabled() {
				return fmt.Errorf("unknown config-type %q", str)
			}
			c.Type = config.ServiceLevelObjective{}
		default:
			c.Type = config.ClassicApiType{Api: str}
		}
		return nil
	}

	// If the shorthand is not used, we need to unmarshal into the more complex map and unmarshal it later into the specific types.
	var data map[string]any
	if err := unmarshal(&data); err != nil {
		return fmt.Errorf("failed to unmarshal type definition: %w", err)
	}

	// Exactly one type must be set, and only from an allowed pool of keys.
	types := maps.Keys(data)
	if len(types) >= 2 {
		return errors.New("only one config type is allowed at once")
	}
	if len(types) == 0 {
		return errors.New("no type is defined")
	}

	ttype := types[0]

	// Now we know the one type and can call the unmarshalers.
	// The unmarshalers write to the type directly to update it, which is a design choice, not a requirement.
	unmarshalers := map[string]func(data any) error{
		"api":        c.parseApiType,
		"settings":   c.parseSettingsType,
		"automation": c.parseAutomation,
		"document":   c.parseDocumentType,
	}

	if featureflags.OpenPipeline.Enabled() {
		unmarshalers["openpipeline"] = c.parseOpenPipelineType
	}

	if unm, f := unmarshalers[ttype]; !f {
		return fmt.Errorf("unknown config-type %q", ttype)
	} else {
		return unm(data[ttype])
	}
}

func (c *TypeDefinition) parseApiType(a any) error {
	// shorthand
	if str, ok := a.(string); ok {
		c.Type = config.ClassicApiType{Api: str}
		return nil
	}

	// full definition
	var r ComplexApiDefinition
	err := mapstructure.Decode(a, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal api-type: %w", err)
	}

	c.Type = config.ClassicApiType{Api: r.Name}
	c.Scope = r.Scope
	return nil
}

func (c *TypeDefinition) parseSettingsType(a any) error {
	var r SettingsDefinition
	err := mapstructure.Decode(a, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal settings-type: %w", err)
	}

	if !featureflags.AccessControlSettings.Enabled() && r.Permissions != nil {
		return fmt.Errorf("unknown settings configuration property 'permissions'")
	}

	var allUserPermission config.AllUserPermissionKind
	if r.Permissions != nil {
		allUserPermission = *r.Permissions.AllUsers
	}
	c.Type = config.SettingsType{
		SchemaId:          r.Schema,
		SchemaVersion:     r.SchemaVersion,
		AllUserPermission: allUserPermission,
	}
	c.Scope = r.Scope
	c.InsertAfter = r.InsertAfter
	return nil
}

func (c *TypeDefinition) parseAutomation(a any) error {
	var r AutomationDefinition
	err := mapstructure.Decode(a, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal automation-type: %w", err)
	}

	c.Type = config.AutomationType{Resource: r.Resource}

	return nil
}

func (c *TypeDefinition) parseDocumentType(a any) error {
	var r DocumentDefinition
	err := mapstructure.Decode(a, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal document-type: %w", err)
	}

	c.Type = config.DocumentType{
		Kind:    r.Kind,
		Private: r.Private,
	}

	return nil
}

func (c *TypeDefinition) parseOpenPipelineType(a any) error {
	var r OpenPipelineDefinition
	err := mapstructure.Decode(a, &r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal openpipeline-type: %w", err)
	}

	c.Type = config.OpenPipelineType{
		Kind: r.Kind,
	}

	return nil
}

// Validate verifies whether the given type definition is valid (correct APIs, fields set, etc)
func (c *TypeDefinition) Validate(apis map[string]struct{}) error {
	switch t := c.Type.(type) {
	case config.ClassicApiType:
		if _, f := apis[t.Api]; !f {
			return fmt.Errorf("unknown API: %s", t.Api)
		}

	case config.SettingsType:
		if t.SchemaId == "" {
			return errors.New("missing settings schemaId")
		}

		if c.Scope == nil {
			return errors.New("missing settings scope")
		}

		if featureflags.AccessControlSettings.Enabled() {
			if t.AllUserPermission != "" && !slices.Contains(config.KnownAllUserPermissionKind, t.AllUserPermission) {
				return fmt.Errorf("unknown all-users value: '%s', allowed: %v", t.AllUserPermission, config.KnownAllUserPermissionKind)
			}
		}

	case config.AutomationType:
		switch t.Resource {
		case "":
			return errors.New("missing automation resource property")

		case config.Workflow, config.BusinessCalendar, config.SchedulingRule:
			return nil

		default:
			return fmt.Errorf("unknown automation resource %q", t.Resource)
		}

	case config.DocumentType:

		if t.Kind == "" {
			return errors.New("missing document kind property")
		}

		if slices.Contains(config.KnownDocumentKinds, t.Kind) {
			return nil
		}

		return fmt.Errorf("unknown document kind %q", t.Kind)

	case config.OpenPipelineType:
		if t.Kind == "" {
			return errors.New("missing openpipeline kind property")
		}
	}

	return nil
}

func (c *TypeDefinition) GetApiType() string {
	switch t := c.Type.(type) {
	case config.ClassicApiType:
		return t.Api
	case config.SettingsType:
		return t.SchemaId
	case config.AutomationType:
		return string(t.Resource)
	case config.BucketType:
		return string(t.ID())
	case config.DocumentType:
		return string(t.ID())
	case config.OpenPipelineType:
		return string(t.ID())
	case config.Segment:
		return string(t.ID())
	case config.ServiceLevelObjective:
		return string(t.ID())
	}

	return ""
}

func (c TypeDefinition) MarshalYAML() (interface{}, error) {
	switch t := c.Type.(type) {

	case config.ClassicApiType:
		// if the scope is empty we can return the simple object.
		if c.Scope == nil {
			return map[string]string{
				"api": t.Api,
			}, nil
		}

		return map[string]any{
			"api": ComplexApiDefinition{
				Name:  t.Api,
				Scope: c.Scope,
			},
		}, nil

	case config.SettingsType:
		var insertAfterValue ConfigParameter
		if featureflags.PersistSettingsOrder.Enabled() {
			insertAfterValue = c.InsertAfter
		}

		setDefinition := SettingsDefinition{
			Schema:        t.SchemaId,
			SchemaVersion: t.SchemaVersion,
			Scope:         c.Scope,
			InsertAfter:   insertAfterValue,
		}

		if featureflags.AccessControlSettings.Enabled() {
			if t.AllUserPermission != "" {
				setDefinition.Permissions = &PermissionDefinition{
					AllUsers: &t.AllUserPermission,
				}
			}
		}

		return map[string]any{"settings": setDefinition}, nil

	case config.AutomationType:
		return map[string]any{
			"automation": AutomationDefinition{
				Resource: t.Resource,
			},
		}, nil

	case config.BucketType:
		return BucketType, nil

	case config.DocumentType:
		return map[string]any{
			"document": DocumentDefinition{
				Kind:    t.Kind,
				Private: t.Private,
			},
		}, nil

	case config.OpenPipelineType:
		if featureflags.OpenPipeline.Enabled() {
			return map[string]any{
				"openpipeline": OpenPipelineDefinition{
					Kind: t.Kind,
				},
			}, nil
		}

	case config.Segment:
		if featureflags.Segments.Enabled() {
			return SegmentType, nil
		}

	case config.ServiceLevelObjective:
		if featureflags.ServiceLevelObjective.Enabled() {
			return ServiceLevelObjectiveType, nil
		}
	}
	return nil, fmt.Errorf("unknown type: %T", c.Type)
}
