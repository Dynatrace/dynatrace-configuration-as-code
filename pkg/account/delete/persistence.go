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

package delete

import (
	"github.com/invopop/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"

	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
)

type (
	FileDefinition struct {
		DeleteEntries []any `yaml:"delete"`
	}

	// DeleteEntry defines the one shared property of account delete entries - their Type
	// Individual entries are to be loaded as UserDeleteEntry, GroupDeleteEntry or PolicyDeleteEntry nased on the content of Type
	DeleteEntry struct {
		Type string `yaml:"type" json:"type" mapstructure:"type" jsonschema:"required,enum=user,enum=group,enum=policy"`
	}
	UserDeleteEntry struct {
		Email string `mapstructure:"email"`
	}
	ServiceUserDeleteEntry struct {
		Name string `mapstructure:"name"`
	}
	GroupDeleteEntry struct {
		Name string `mapstructure:"name"`
	}
	PolicyDeleteEntry struct {
		Name  string      `mapstructure:"name"`
		Level PolicyLevel `mapstructure:"level"` // either PolicyLevelAccount or PolicyLevelEnvironment
	}
	PolicyLevel struct {
		Type        string `mapstructure:"type"`
		Environment string `mapstructure:"environment"`
	}

	SchemaDef struct {
		DeleteEntries Entries `json:"delete" jsonschema:"required"`
	}
	Entries []DeleteEntry
)

// JSONSchema manually defines the schema for account DeleteEntry as the nature of this structs dependent required
// fields makes it impossible to simply generate the schema via reflection.
// This definition likely needs to change if the DeleteEntry changes
func (_ Entries) JSONSchema() *jsonschema.Schema {
	base := jsonutils.ReflectJSONSchema(DeleteEntry{})

	props := base.Properties
	props.Set("email", &jsonschema.Schema{Type: "string", Description: "email of the user to delete - required for type user"})
	props.Set("name", &jsonschema.Schema{Type: "string", Description: "name of the group, policy or service user to delete"})

	props.Set("level", &jsonschema.Schema{
		Type:        "object",
		Description: "level of policy to delete",
		Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Type: "string", Description: "level type of the policy to delete", Enum: []any{"account", "environment"}}},
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "environment", Value: &jsonschema.Schema{Type: "string", Description: "environment to delete the policy for"}})),
		Required: []string{"type"},
		OneOf: []*jsonschema.Schema{
			{
				Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
					orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "environment"}})),
				Required: []string{"environment"},
			},
			{
				Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
					orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "account"}})),
			},
		},
	})

	conditionalRequiredFields := make([]*jsonschema.Schema, 0)
	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "user"}})),
		Required: []string{"email"},
	})

	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "service-user"}})),
		Required: []string{"name"},
	})

	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "group"}})),
		Required: []string{"name"},
	})

	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: orderedmap.New[string, *jsonschema.Schema](orderedmap.WithInitialData(
			orderedmap.Pair[string, *jsonschema.Schema]{Key: "type", Value: &jsonschema.Schema{Const: "policy"}})),
		Required: []string{"name", "level"},
	})

	return &jsonschema.Schema{
		Type: "array",
		Items: &jsonschema.Schema{
			Type:                 base.Type,
			Properties:           base.Properties,
			AdditionalProperties: base.AdditionalProperties,
			Required:             base.Required,
			Comments:             base.Comments,
			AnyOf:                conditionalRequiredFields,
		},
	}
}
