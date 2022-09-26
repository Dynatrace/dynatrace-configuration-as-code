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
	"strings"

	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter/reference"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
)

// ResolveDependencies resolves all id-dependencies between downloaded configs.
//
// We do this by collecting all ids of all configs, and then simply by searching for them in templates.
// If we find an occurrence, we replace it with a generic variable and reference the config.
func ResolveDependencies(configs project.ConfigsPerApis) project.ConfigsPerApis {
	log.Debug("Resolving dependencies between configs")

	configsById := collectConfigsById(configs)

	findAndSetDependencies(configs, configsById)

	log.Debug("Finished resolving dependencies")

	return configs
}

func findAndSetDependencies(configs project.ConfigsPerApis, configsById map[string]config.Config) {
	// currently a simple brute force attach, could be parallelized
	for theApi, configs := range configs {
		for _, configToBeUpdated := range configs {
			newContent, parameters := findAndReplaceIds(theApi, configToBeUpdated, configsById)

			maps.Copy(configToBeUpdated.Parameters, parameters)
			configToBeUpdated.Template.UpdateContent(newContent)
		}
	}
}

func collectConfigsById(configs project.ConfigsPerApis) map[string]config.Config {
	configsById := map[string]config.Config{}

	for _, configs := range configs {
		for _, conf := range configs {
			configsById[conf.Template.Id()] = conf
		}
	}
	return configsById
}

func findAndReplaceIds(apiName string, configToBeUpdated config.Config, configs map[string]config.Config) (string, config.Parameters) {
	parameters := make(config.Parameters, 0)
	content := configToBeUpdated.Template.Content()

	for key, conf := range configs {
		if strings.Contains(content, key) && conf.Template.Id() != configToBeUpdated.Template.Id() {
			log.Debug("\treference: '%v/%v' referencing '%v' in coordinate '%v' ", apiName, configToBeUpdated.Template.Id(), key, conf.Coordinate)

			parameterName := util.SanitizeTemplateVar(fmt.Sprintf("%v__%v__id", conf.Coordinate.Api, conf.Coordinate.Config))
			coord := conf.Coordinate

			content = strings.ReplaceAll(content, key, "{{."+parameterName+"}}")
			ref := reference.NewWithCoordinate(coord, "id")
			parameters[parameterName] = ref
		}
	}

	return content, parameters
}
