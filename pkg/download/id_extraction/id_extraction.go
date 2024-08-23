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

package id_extraction

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	ref "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"regexp"
	"strings"
)

// meIDRegexPattern matching a Dynatrace Monitored Entity ID which consists of a type containing characters and
// underscores, a dash separator '-' and 16 hex numbers
var meIDRegexPattern = regexp.MustCompile(`[a-zA-Z_]+-[A-Fa-f0-9]{16}`)

var uuidRegexPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

const baseParamID = "extractedIDs"

// ExtractIDsIntoYAML searches for Dynatrace ID patterns in each given config and extracts them from the config's
// JSON template, into a YAML parameter. It modifies the given configsPerType map.
func ExtractIDsIntoYAML(configsPerType project.ConfigsPerType) (project.ConfigsPerType, error) {
	for _, cfgs := range configsPerType {
		for _, c := range cfgs {
			content, err := c.Template.Content()
			if err != nil {
				return nil, fmt.Errorf("failed to extract IDs from %s: %w", c.Coordinate, err)
			}

			ids := findAllIds(content)

			idMap := map[string]string{}

			for _, id := range ids {
				idKey := createParameterKey(id)

				if _, exists := idMap[idKey]; exists {
					continue // no need to re-add an ID that was found several times in the template
				}

				idMap[idKey] = id

				paramID := fmt.Sprintf("{{ .%s.%s }}", baseParamID, idKey)

				content = strings.ReplaceAll(content, id, paramID)
			}

			if featureflags.Permanent[featureflags.ExtractScopeAsParameter].Enabled() {
				scopeParam := c.Parameters[config.ScopeParameter]
				if scopeParam != nil && scopeParam.GetType() == value.ValueParameterType {
					scopeParamResolved, err := scopeParam.ResolveValue(parameter.ResolveContext{})
					if err != nil {
						return nil, fmt.Errorf("failed to resolve scope paramter from %s: %w", c.Coordinate, err)
					}
					if scopeParamResolved != "environment" {
						scopeParamRsolvedStr := scopeParamResolved.(string)
						key := createParameterKey(scopeParamRsolvedStr)
						idMap[key] = scopeParamRsolvedStr
						c.Parameters[config.ScopeParameter] = &ref.ReferenceParameter{
							ParameterReference: parameter.ParameterReference{Config: c.Coordinate, Property: baseParamID + "." + key},
						}
					}
				}
			}

			if len(idMap) > 0 { // found IDs, update template with new content and store to parameters
				err = c.Template.UpdateContent(content)
				if err != nil {
					return nil, fmt.Errorf("failed to extract IDs from %s: %w", c.Coordinate, err)
				}
				c.Parameters[baseParamID] = value.New(idMap)
			}
		}
	}
	return configsPerType, nil
}

func findAllIds(content string) []string {
	ids := meIDRegexPattern.FindAllString(content, -1)
	ids = append(ids, uuidRegexPattern.FindAllString(content, -1)...)
	return ids
}

func createParameterKey(id string) string {
	idKey := strings.ReplaceAll(id, "-", "_")   // golang template keys must not contain hyphens
	idKey = strings.ReplaceAll(idKey, ".", "_") // monaco templating would treat any dot as referencing a nested sub-key in value parameters, but we're just building simple key:val parameters
	return fmt.Sprintf("id_%s", idKey)
}
