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

package json

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/invopop/jsonschema"
)

func GenerateJSONSchemaString(value interface{}) ([]byte, error) {
	log.Debug("Generating JSON schema for %T...", value)

	s := ReflectJSONSchema(value)

	b, err := s.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON schema: %w", err)
	}

	return MarshalIndent(b), nil
}

func ReflectJSONSchema(value interface{}) *jsonschema.Schema {
	r := new(jsonschema.Reflector)
	r.RequiredFromJSONSchemaTags = true // not all our optional fields have a json 'omitempty' tag, so we tag required explicitly
	r.DoNotReference = true
	err := r.AddGoComments("github.com/dynatrace/dynatrace-configuration-as-code/v2", ".")
	if err != nil {
		log.Warn("Failed to parse Go comments, schema descriptions may be incomplete")
	}
	return r.Reflect(value)
}
