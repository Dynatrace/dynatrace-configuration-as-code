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
	"github.com/cloudflare/ahocorasick"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/reference"
	"strings"
)

// dependencyResolutionContext holds the important information for dependency resolution.
// The core information is in the [idMatcher] that returns the index of the found strings in the initial dictionary.
// Since not the ids, but the indexes are returned, we also need the original dictionary.
// - [ids] to look up the actual content of the key,
// - and with this key we can look up the actual config using the [configsById] field.
type dependencyResolutionContext struct {
	// the original dictionary that initialized the [idMatcher].
	// it is used to get the actual key after the matcher returns the index(es)
	ids []string

	// idMatcher is the matcher that returns all found strings within a searched string as the index of the dictionary
	idMatcher *ahocorasick.Matcher

	// configsById holds all configs by their id. It is used to get the config for a given key
	configsById map[string]config.Config
}

type ahocorasickResolver struct {
	ctx dependencyResolutionContext
}

func AhocorasickResolver(configsById map[string]config.Config) ahocorasickResolver {
	ids := maps.Keys(configsById)

	ctx := dependencyResolutionContext{
		ids:         ids,
		idMatcher:   ahocorasick.NewStringMatcher(ids),
		configsById: configsById,
	}

	return ahocorasickResolver{
		ctx: ctx,
	}
}

func (r ahocorasickResolver) ResolveDependencyReferences(configToBeUpdated *config.Config) {
	resolveScope(configToBeUpdated, r.ctx.configsById)
	resolveTemplate(configToBeUpdated, r.ctx)
}

func resolveTemplate(configToBeUpdated *config.Config, c dependencyResolutionContext) {
	newContent, parameters, _ := findAndReplaceIDs(configToBeUpdated.Coordinate.Type, *configToBeUpdated, c)

	maps.Copy(configToBeUpdated.Parameters, parameters)
	configToBeUpdated.Template.UpdateContent(newContent)
}

func findAndReplaceIDs(apiName string, configToBeUpdated config.Config, c dependencyResolutionContext) (string, config.Parameters, []coordinate.Coordinate) {
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

		parameterName := CreateParameterName(conf.Coordinate.Type, conf.Coordinate.ConfigId)
		coord := conf.Coordinate

		content = strings.ReplaceAll(content, key, "{{."+parameterName+"}}")
		ref := reference.NewWithCoordinate(coord, "id")
		parameters[parameterName] = ref
		coordinates = append(coordinates, coord)

	}

	return content, parameters, coordinates
}
