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
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/iancoleman/orderedmap"
	"github.com/invopop/jsonschema"
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
	props.Set("email", map[string]any{"type": "string", "description": "email of the user to delete - required for type user"})
	props.Set("name", map[string]any{"type": "string", "description": "name of the group or policy to delete"})
	props.Set("level", map[string]any{"type": "object", "description": "level of policy to delete",
		"properties": map[string]any{
			"type":        map[string]any{"type": "string", "description": "level type of the policy to delete", "enum": []string{"account", "environment"}},
			"environment": map[string]any{"type": "string", "description": "environment to delete the policy for"},
		},
		"required": []string{"type"},
		"oneOf": []map[string]any{
			{
				"properties": map[string]any{
					"type": map[string]any{"const": "environment"},
				},
				"required": []string{"environment"},
			},
			{
				"properties": map[string]any{
					"type": map[string]any{"const": "account"},
				},
			},
		},
	})

	conditionalRequiredFields := make([]*jsonschema.Schema, 0)
	opts := orderedmap.New()
	opts.Set("type", map[string]any{"const": "user"})
	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: opts,
		Required:   []string{"email"},
	})

	opts = orderedmap.New()
	opts.Set("type", map[string]any{"const": "group"})
	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: opts,
		Required:   []string{"name"},
	})

	opts = orderedmap.New()
	opts.Set("type", map[string]any{"const": "policy"})
	conditionalRequiredFields = append(conditionalRequiredFields, &jsonschema.Schema{
		Properties: opts,
		Required:   []string{"name", "level"},
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
