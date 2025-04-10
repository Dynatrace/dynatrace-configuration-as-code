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

	goaho "github.com/anknown/ahocorasick"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
)

// dependencyResolutionContext holds the important information for dependency resolution.
// The core information is in the [matcher] that returns the index of the found strings in the initial dictionary.
// As this will just return the string IDs of configurations, the context also holds the [configsById] map, which is
// used to access the actual config data if a match was found.
type dependencyResolutionContext struct {
	// matcher is the matcher that returns all found strings within a searched string
	matcher *goaho.Machine

	// configsById holds all configs by their id. It is used to get the config for a given key
	configsById map[string]config.Config
}

type ahocorasickResolver struct {
	ctx dependencyResolutionContext
}

func AhoCorasickResolver(configsById map[string]config.Config) (ahocorasickResolver, error) {

	ids := toRuneSlices(maps.Keys(configsById))
	m := &goaho.Machine{}
	err := m.Build(ids)
	if err != nil {
		return ahocorasickResolver{}, fmt.Errorf("failed to initialize AhoCorasick matcher: %w", err)
	}

	ctx := dependencyResolutionContext{
		matcher:     m,
		configsById: configsById,
	}

	return ahocorasickResolver{
		ctx: ctx,
	}, nil
}

// toRuneSlices converts a given slice of string config ids to a slice of rune slices
// this is the format the aho-corasick implementation expects to receive search keys in
func toRuneSlices(ids []string) [][]rune {
	dict := make([][]rune, len(ids))
	for i, s := range ids {
		dict[i] = []rune(s)
	}
	return dict
}

func (r ahocorasickResolver) ResolveDependencyReferences(configToBeUpdated *config.Config) error {
	resolveScope(configToBeUpdated, r.ctx.configsById)
	return resolveTemplate(configToBeUpdated, r.ctx)
}

func resolveTemplate(configToBeUpdated *config.Config, c dependencyResolutionContext) error {
	newContent, parameters, err := findAndReplaceIDs(configToBeUpdated.Coordinate.Type, *configToBeUpdated, c)
	if err != nil {
		return err
	}

	maps.Copy(configToBeUpdated.Parameters, parameters)
	return configToBeUpdated.Template.UpdateContent(newContent)
}

func findAndReplaceIDs(apiName string, configToBeUpdated config.Config, c dependencyResolutionContext) (string, config.Parameters, error) {
	parameters := make(config.Parameters, 0)
	content, err := configToBeUpdated.Template.Content()
	if err != nil {
		return "", nil, err
	}

	matches := c.matcher.MultiPatternSearch([]rune(content), false)
	for _, m := range matches {

		key := string(m.Word)

		// get the actual key and config for a given match
		conf, f := c.configsById[key]
		if !f {
			panic(fmt.Sprintf("No config found for given key %q", key))
		}

		if !canReference(configToBeUpdated, conf) {
			continue
		}

		log.Debug("\treference: '%v/%v' referencing '%v' in coordinate '%v' ", apiName, configToBeUpdated.Template.ID(), key, conf.Coordinate)

		parameterName := CreateParameterName(conf.Coordinate.Type, conf.Coordinate.ConfigId)

		newContent := replaceAll(content, key, "{{."+parameterName+"}}")
		if newContent != content {
			parameters[parameterName] = reference.NewWithCoordinate(conf.Coordinate, "id")
			content = newContent
		}
	}

	return content, parameters, nil
}
