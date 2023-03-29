/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package download

import (
	"fmt"
	"github.com/cloudflare/ahocorasick"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	valueParam "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"regexp"
	"strings"
	"sync"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

// ResolveDependencies resolves all id-dependencies between downloaded configs.
//
// We do this by collecting all ids of all configs, and then simply by searching for them in templates.
// If we find an occurrence, we replace it with a generic variable and reference the config.
func ResolveDependencies(configs project.ConfigsPerType) project.ConfigsPerType {
	log.Debug("Resolving dependencies between configs")
	resolve(configs)
	log.Debug("Finished resolving dependencies")
	return configs
}

// dependencyResolutionContext holds the important information for depdenency resolution.
// The core information is in the [idMatcher] that returns the index of the found strings in the initial dictionary.
// Since not the ids, but the indexes are returned, we need also the original dictionary ([ids] to look up the actual content of the key,
// and with this key we can look up the actual config using the [configsById] field.
type dependencyResolutionContext struct {
	// the original dictionary that initialized the [idMatcher].
	// it is used to get the actual key after the matcher returns the index(es)
	ids []string

	// idMatcher is the matcher that returns all found strings within a searched string as the index of the dictionary
	idMatcher *ahocorasick.Matcher

	// configsById holds all configs by their id. It is used to get the config for a given key
	configsById map[string]config.Config
}

func resolve(configs project.ConfigsPerType) {
	c := newResolutionContext(configs)

	wg := sync.WaitGroup{}
	// currently a simple brute force attach
	for _, configs := range configs {
		configs := configs
		for i := range configs {
			wg.Add(1)

			configToBeUpdated := &configs[i]
			go func() {
				resolveScope(configToBeUpdated, c.configsById)
				resolveTemplate(configToBeUpdated, c)

				wg.Done()
			}()
		}
	}

	wg.Wait()
}

func newResolutionContext(configs project.ConfigsPerType) dependencyResolutionContext {
	configsById := collectConfigsById(configs)
	ids := maps.Keys(configsById)

	return dependencyResolutionContext{
		ids:         ids,
		idMatcher:   ahocorasick.NewStringMatcher(ids),
		configsById: configsById,
	}
}

func resolveScope(configToBeUpdated *config.Config, ids map[string]config.Config) {
	if configToBeUpdated.Type.ID() != config.SettingsTypeId {
		return
	}

	scopeParam, found := configToBeUpdated.Parameters[config.ScopeParameter]
	if !found {
		log.Error(fmt.Sprintf("Setting found without a scope parameter. Skipping resolution for this config. Coordinate: %s.", configToBeUpdated.Coordinate))
		return
	}

	value, ok := scopeParam.(*valueParam.ValueParameter)
	if scopeParam.GetType() != valueParam.ValueParameterType || !ok {
		log.Error(fmt.Sprintf("Expected scope parameter to be a value. Skipping resolution for this config. Coordinate: %s.", configToBeUpdated.Coordinate))
		return
	}

	dependency, found := ids[fmt.Sprint(value.Value)]
	if !found {
		return
	}

	configToBeUpdated.Parameters[config.ScopeParameter] = reference.NewWithCoordinate(dependency.Coordinate, "id")
}

func resolveTemplate(configToBeUpdated *config.Config, c dependencyResolutionContext) {
	newContent, parameters, _ := findAndReplaceIds(configToBeUpdated.Coordinate.Type, *configToBeUpdated, c)

	maps.Copy(configToBeUpdated.Parameters, parameters)
	configToBeUpdated.Template.UpdateContent(newContent)
}

func collectConfigsById(configs project.ConfigsPerType) map[string]config.Config {
	configsById := map[string]config.Config{}

	for _, configs := range configs {
		for _, conf := range configs {
			configsById[conf.Template.Id()] = conf
		}
	}
	return configsById
}

func findAndReplaceIds(apiName string, configToBeUpdated config.Config, c dependencyResolutionContext) (string, config.Parameters, []coordinate.Coordinate) {
	parameters := make(config.Parameters, 0)
	content := configToBeUpdated.Template.Content()
	coordinates := make([]coordinate.Coordinate, 0)

	indexes := c.idMatcher.MatchThreadSafe([]byte(configToBeUpdated.Template.Content()))
	for _, v := range indexes {

		// get the actual key and config for a given match
		key := c.ids[v]
		conf, f := c.configsById[key]
		if !f {
			panic(fmt.Sprintf("No config found for given key %q", key))
		}

		if configToBeUpdated.Coordinate.Type == "dashboard" && conf.Coordinate.Type == "dashboard" {
			continue // dashboards can not actually reference each other, but often contain a link to another inside a markdown tile
		}

		if conf.Coordinate == configToBeUpdated.Coordinate {
			continue // skip self referencing configs
		}

		log.Debug("\treference: '%v/%v' referencing '%v' in coordinate '%v' ", apiName, configToBeUpdated.Template.Id(), key, conf.Coordinate)

		parameterName := createParameterName(conf.Coordinate.Type, conf.Coordinate.ConfigId)
		coord := conf.Coordinate

		content = strings.ReplaceAll(content, key, "{{."+parameterName+"}}")
		ref := reference.NewWithCoordinate(coord, "id")
		parameters[parameterName] = ref
		coordinates = append(coordinates, coord)

	}

	return content, parameters, coordinates
}

func createParameterName(api, configId string) string {
	return sanitizeTemplateVar(fmt.Sprintf("%v__%v__id", api, configId))
}

// matches any non-alphanumerical chars including _
var templatePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// SanitizeTemplateVar removes all except alphanumerical chars and underscores (_)
func sanitizeTemplateVar(templateVarName string) string {
	return templatePattern.ReplaceAllString(templateVarName, "")
}
