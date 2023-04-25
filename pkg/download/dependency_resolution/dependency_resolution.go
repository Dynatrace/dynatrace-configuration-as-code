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

package dependency_resolution

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/dependency_resolution/resolver"
	"sync"

	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

type dependencyResolver interface {
	ResolveDependencyReferences(configToBeUpdated *config.Config)
}

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

func resolve(configs project.ConfigsPerType) {
	r := getResolver(configs)

	wg := sync.WaitGroup{}
	// currently a simple brute force attach
	for _, configs := range configs {
		configs := configs
		for i := range configs {
			wg.Add(1)

			configToBeUpdated := &configs[i]
			go func() {
				r.ResolveDependencyReferences(configToBeUpdated)

				wg.Done()
			}()
		}
	}

	wg.Wait()
}

func getResolver(configs project.ConfigsPerType) dependencyResolver {
	configsById := collectConfigsById(configs)

	if featureflags.FastDependencyResolver().Enabled() {
		log.Debug("Using fast but memory intensive dependency resolution. Can be deactivated using '%s=false' env var.", featureflags.FastDependencyResolver().EnvName())
		return resolver.AhocorasickResolver(configsById)
	}
	log.Debug("Using slow (CPU intensive) but memory saving dependency resolution.")
	return resolver.BasicResolver(configsById)
}

func collectConfigsById(configs project.ConfigsPerType) map[string]config.Config {
	configsById := map[string]config.Config{}

	for _, configs := range configs {
		for _, conf := range configs {
			configsById[conf.Coordinate.ConfigId] = conf
			if conf.OriginObjectId != "" {
				// resolve Settings references by Object ID as well
				configsById[conf.OriginObjectId] = conf
			}
			if conf.OriginObjectId != "" && conf.Coordinate.Type == "builtin:management-zones" {
				// resolve Management Zone Settings by Numeric ID as well
				numID, err := idutils.GetNumericIDForObjectID(conf.OriginObjectId)
				if err != nil {
					log.Warn("Failed to decode numeric ID of config %q, auto-resolved references may be incomplete: %v", conf.Coordinate, err)
				} else {
					strId := fmt.Sprintf("%d", numID)
					configsById[strId] = conf
				}
			}
		}
	}
	return configsById
}
