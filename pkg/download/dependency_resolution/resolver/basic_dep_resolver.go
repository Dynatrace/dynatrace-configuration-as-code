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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"golang.org/x/exp/maps"
	"strings"
)

type basicResolver struct {
	configsById map[string]config.Config
}

func BasicResolver(configsById map[string]config.Config) basicResolver {
	return basicResolver{
		configsById: configsById,
	}
}

func (r basicResolver) ResolveDependencyReferences(configToBeUpdated *config.Config) error {
	resolveScope(configToBeUpdated, r.configsById)
	return basicResolveTemplate(configToBeUpdated, r.configsById)
}

func basicResolveTemplate(configToBeUpdated *config.Config, configsById map[string]config.Config) error {
	newContent, parameters, err := basicFindAndReplaceIDs(configToBeUpdated.Coordinate.Type, *configToBeUpdated, configsById)
	if err != nil {
		return err
	}

	maps.Copy(configToBeUpdated.Parameters, parameters)
	return configToBeUpdated.Template.UpdateContent(newContent)
}

func basicFindAndReplaceIDs(apiName string, configToBeUpdated config.Config, configs map[string]config.Config) (string, config.Parameters, error) {
	parameters := make(config.Parameters, 0)
	content, err := configToBeUpdated.Template.Content()
	if err != nil {
		return "", nil, err
	}

	for key, conf := range configs {
		if shouldReplaceReference(configToBeUpdated, conf, content, key) {
			log.Debug("\treference: '%v/%v' referencing '%v' in coordinate '%v' ", apiName, configToBeUpdated.Template.Id(), key, conf.Coordinate)

			parameterName := CreateParameterName(conf.Coordinate.Type, conf.Coordinate.ConfigId)
			coord := conf.Coordinate

			content = strings.ReplaceAll(content, key, "{{."+parameterName+"}}")
			ref := reference.NewWithCoordinate(coord, "id")
			parameters[parameterName] = ref
		}
	}

	return content, parameters, nil
}

// shouldReplaceReference checks if a given key is found in the content of another config and should be replaced
// in case two configs are actually the same, or if they are both dashboards no replacement will happen as in these
// cases there is no real valid reference even if the key is found in the content.
func shouldReplaceReference(configToBeUpdated config.Config, configToUpdateFrom config.Config, contentToBeUpdated, keyToReplace string) bool {
	if configToBeUpdated.Coordinate.Type == "dashboard" && configToUpdateFrom.Coordinate.Type == "dashboard" {
		return false //dashboards can not actually reference each other, but often contain a link to another inside a markdown tile
	}

	return configToUpdateFrom.Template.Id() != configToBeUpdated.Template.Id() && strings.Contains(contentToBeUpdated, keyToReplace)
}
