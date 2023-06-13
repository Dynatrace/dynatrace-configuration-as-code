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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"regexp"
	"strings"
)

// meIDRegexPattern matching a Dynatrace Monitored Entity ID which consists of a type containing characters and
// underscores, a dash separator '-' and a 16 char alphanumeric ID
var meIDRegexPattern = regexp.MustCompile(`[a-zA-Z_]+-[A-Za-z0-9]{16}`)

var uuidRegexPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

const baseParamID = "extractedIDs"

// ExtractIDsIntoYAML searches for Dynatrace ID patterns in each given config and extracts them from the config's
// JSON template, into a YAML parameter. It modifies the given configsPerType map.
func ExtractIDsIntoYAML(configsPerType project.ConfigsPerType) project.ConfigsPerType {
	for _, cfgs := range configsPerType {
		for _, c := range cfgs {
			ids := meIDRegexPattern.FindAllString(c.Template.Content(), -1)
			ids = append(ids, uuidRegexPattern.FindAllString(c.Template.Content(), -1)...)

			idMap := map[string]string{}

			for _, id := range ids {
				idKey := createParameterKey(id)

				if _, exists := idMap[idKey]; exists {
					continue // no need to re-add an ID that was found several times in the template
				}

				idMap[idKey] = id

				paramID := fmt.Sprintf("{{ .%s.%s }}", baseParamID, idKey)

				newContent := strings.ReplaceAll(c.Template.Content(), id, paramID)
				c.Template.UpdateContent(newContent)
			}

			if len(idMap) > 0 {
				c.Parameters[baseParamID] = value.New(idMap)
			}
		}
	}
	return configsPerType
}

func createParameterKey(id string) string {
	idKey := strings.ReplaceAll(id, "-", "_") // golang template keys must not contain hyphens
	return fmt.Sprintf("id_%s", idKey)
}
