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

package resolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
)

// resolveScope updates the `scope` parameter of the config and converts it to a reference parameter iff the scope
// is a known id of another downloaded config.
func resolveScope(configToBeUpdated *config.Config, ids map[string]config.Config) {
	if configToBeUpdated.Type.ID() != config.SettingsTypeID {
		return
	}

	scopeParam, found := configToBeUpdated.Parameters[config.ScopeParameter]
	if !found {
		log.Error("Setting found without a scope parameter. Skipping resolution for this config. Coordinate: %s.", configToBeUpdated.Coordinate)
		return
	}

	value, ok := scopeParam.(*valueParam.ValueParameter)
	if scopeParam.GetType() != valueParam.ValueParameterType || !ok {
		log.Error("Expected scope parameter to be a value. Skipping resolution for this config. Coordinate: %s.", configToBeUpdated.Coordinate)
		return
	}

	dependency, found := ids[fmt.Sprint(value.Value)]
	if !found {
		return
	}

	configToBeUpdated.Parameters[config.ScopeParameter] = reference.NewWithCoordinate(dependency.Coordinate, "id")
}

func CreateParameterName(api, configId string) string {
	return sanitizeTemplateVar(fmt.Sprintf("%v__%v__id", api, configId))
}

// matches any non-alphanumerical chars including _
var templatePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// SanitizeTemplateVar removes all except alphanumerical chars and underscores (_)
func sanitizeTemplateVar(templateVarName string) string {
	return templatePattern.ReplaceAllString(templateVarName, "")
}

func replaceAll(content string, key string, s string) string {
	// The prefix and suffix we search for are alphanumerical, as well as the "-", and "_".
	// From investigating, this character set seems to be the most basic regex that still avoids false positive substring matches.
	str := fmt.Sprintf("([^a-zA-Z0-9_-])(%s)([^a-zA-Z0-9_-])", key)

	// replace only strings that are not part of another larger string. See testcases for exact in/out values.
	re, err := regexp.Compile(str)
	if err != nil {
		log.Debug("Failed to compile string %q to regex. Falling back to use simple string replace.", str)
		return strings.ReplaceAll(content, key, s)
	}

	return re.ReplaceAllString(content, fmt.Sprintf("$1%s$3", s))
}

// canReference verifies whether configToUpdateFrom can actually reference configToBeUpdated.
//
// configToUpdateFrom can not reference configToBeUpdated if either
//   - they are the same config (coordinate matches)
//   - they are both dashboards (remove cyclic dependencies)
//   - configToUpdateFrom is a dashboard-share-setting (can not be referenced)
func canReference(configToBeUpdated config.Config, configToUpdateFrom config.Config) bool {
	if configToBeUpdated.Coordinate == configToUpdateFrom.Coordinate {
		return false // they are the same config
	}

	if configToBeUpdated.Coordinate.Type == "dashboard" && configToUpdateFrom.Coordinate.Type == "dashboard" {
		return false // dashboards can not actually reference each other, but often contain a link to another inside a markdown tile
	}

	if configToUpdateFrom.Coordinate.Type == "dashboard-share-setting" {
		// dashboard share settings can not be referenced, but since they have the same id as their parent dashboard, dashboards suddenly reference them
		return false
	}

	return true
}
