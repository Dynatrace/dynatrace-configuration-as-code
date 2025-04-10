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
	"github.com/invopop/jsonschema"

	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
)

// FileDefinition represents a loaded YAML delete file consisting of a list of delete entries called 'delete'
// In this struct DeleteEntries may either be a legacy shorthand string or full DeleteEntry value.
// Use FullFileDefinition if you're always working with DeleteEntry values instead
type FileDefinition struct {
	// DeleteEntries loaded from a file are either legacy shorthand strings or full DeleteEntry values
	DeleteEntries []interface{} `yaml:"delete"`
}

// FullFileDefinition represents a delete file consisting of a list of delete entries called 'delete'
// In this struct DeleteEntries are DeleteEntry values.
type FullFileDefinition struct {
	// DeleteEntries defining which configurations should be deleted
	DeleteEntries DeleteEntries `yaml:"delete" json:"delete"`
}

// DeleteEntry is a full representation of a delete entry loaded from a YAML delete file
// ConfigId and ConfigName should be mutually exclusive (validated if using LoadEntriesToDelete)
type DeleteEntry struct {
	// Project the config was in - required for configs with generated IDs (e.g. Settings 2.0, Automations, Grail Buckets)
	Project string `yaml:"project,omitempty" json:"project,omitempty" mapstructure:"project" jsonschema:"description=The project the config was in - required for configs with generated IDs (e.g. Settings 2.0, Automations, Grail Buckets)."`
	// Type of the config to be deleted
	Type string `yaml:"type" json:"type" mapstructure:"type" jsonschema:"required,description=The type of config to be deleted."`
	// ConfigId is the monaco ID of the config to be deleted - required for configs with generated IDs (e.g. Settings 2.0, Automations, Grail Buckets)
	ConfigId string `yaml:"id,omitempty" json:"id,omitempty" mapstructure:"id" jsonschema:"description=The monaco ID of the config to be deleted - required for configs with generated IDs (e.g. Settings 2.0, Automations, Grail Buckets). It can't be combined with 'objectId' or 'name'."`
	// ConfigName is the name of the config to be deleted - required for configs deleted by name (classic Config API types)
	ConfigName string `yaml:"name,omitempty" json:"name,omitempty" mapstructure:"name" jsonschema:"description=The name of the config to be deleted - required for configs deleted by name (classic Config API types). It can't be combined with 'objectId' or 'id'."`
	//ObjectId is the dynatrace ID of the object
	ObjectId string `yaml:"objectId,omitempty" json:"objectId,omitempty" mapstructure:"objectId" jsonschema:"ID of the configuration in the Dynatrace. It can't be combined with 'name' or 'id'."`
	// Scope is the parent scope of a config. This field must be set if a classic config is used, and the classic config requires the scope to be set.
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty" mapstructure:"scope" jsonschema:"description=The scope of the config to be deleted - required for API configs that require a scope"`
	// CustomValues holds special values that are not general enough to add as a field to a DeleteEntry but are still important for specific APIs
	CustomValues map[string]string `yaml:",inline" mapstructure:",remain"`
}

type DeleteEntries []DeleteEntry

// JSONSchema defines a custom schema definition for ReferenceSlice as it contains either Reference objects or strings
// when being parsed, but our schema generator can not resolve such a nested "one-of" relation correctly for slices
func (DeleteEntries) JSONSchema() *jsonschema.Schema {
	base := jsonutils.ReflectJSONSchema(DeleteEntry{})

	return &jsonschema.Schema{
		Type: "array",
		Items: &jsonschema.Schema{
			OneOf: []*jsonschema.Schema{
				{
					Type: "string",
				},
				{
					Type: "object",
				},
			},
			Properties:           base.Properties,
			AdditionalProperties: base.AdditionalProperties,
			Required:             base.Required,
			Comments:             base.Comments,
		},
	}
}
