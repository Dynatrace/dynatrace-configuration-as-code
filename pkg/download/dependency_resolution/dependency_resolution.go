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
	"sync"
	"sync/atomic"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution/resolver"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type dependencyResolver interface {
	ResolveDependencyReferences(configToBeUpdated *config.Config) error
}

// ResolveDependencies resolves all id-dependencies between downloaded configs.
//
// We do this by collecting all ids of all configs, and then simply by searching for them in templates.
// If we find an occurrence, we replace it with a generic variable and reference the config.
func ResolveDependencies(configs project.ConfigsPerType) (project.ConfigsPerType, error) {
	log.Debug("Resolving dependencies between configs")
	err := resolve(configs)
	if err != nil {
		return nil, err
	}
	log.Debug("Finished resolving dependencies")
	return configs, nil
}

func resolve(configs project.ConfigsPerType) error {
	r := getResolver(configs)
	errOccurred := atomic.Bool{}
	wg := sync.WaitGroup{}
	// currently a simple brute force approach
	for _, configs := range configs {
		for i := range configs {
			wg.Add(1)

			configToBeUpdated := &configs[i]
			go func() {
				err := r.ResolveDependencyReferences(configToBeUpdated)
				if err != nil {
					log.WithFields(field.Coordinate(configToBeUpdated.Coordinate), field.Error(err)).Error("Failed to resolve dependencies: %v", err)
					errOccurred.Store(true)
				}

				wg.Done()
			}()
		}
	}

	wg.Wait()

	if errOccurred.Load() {
		return fmt.Errorf("failed to resolve dependencies")
	}
	return nil
}

func getResolver(configs project.ConfigsPerType) dependencyResolver {
	configsById := collectConfigsById(configs)

	if featureflags.Permanent[featureflags.FastDependencyResolver].Enabled() {
		log.Debug("Using fast but memory intensive dependency resolution. Can be deactivated by setting the environment variable '%s' to 'false'.", featureflags.Permanent[featureflags.FastDependencyResolver].EnvName())
		r, err := resolver.AhoCorasickResolver(configsById)
		if err != nil {
			log.WithFields(field.Error(err)).Error("Failed to initialize fast dependency resolution, falling back to slow resolution: %v", err)
			return resolver.BasicResolver(configsById)
		}
		return r
	}

	log.Debug("Using slow (CPU intensive) but memory saving dependency resolution.")
	return resolver.BasicResolver(configsById)
}

func collectConfigsById(configs project.ConfigsPerType) map[string]config.Config {
	configsById := map[string]config.Config{}

	for _, configs := range configs {
		for _, conf := range configs {
			// ignore open pipeline configs because their IDs are no UUID or the like.
			// Hence, it is very likely that we replace occurrences that are not meant to represent IDs.
			// e.g. "events" or "logs"
			if conf.Coordinate.Type == string(config.OpenPipelineTypeID) {
				continue
			}
			configsById[conf.Coordinate.ConfigId] = conf
			if conf.OriginObjectId != "" {
				// resolve Settings references by Object ID as well
				configsById[conf.OriginObjectId] = conf
			}
			if conf.OriginObjectId != "" && conf.Coordinate.Type == "builtin:management-zones" {
				// resolve Management Zone Settings by Numeric ID as well
				numID, err := idutils.GetNumericIDForObjectID(conf.OriginObjectId)
				if err != nil {
					log.WithFields(field.Error(err)).Warn("Failed to decode numeric ID of config %q, auto-resolved references may be incomplete: %v", conf.Coordinate, err)
				} else {
					strId := fmt.Sprintf("%d", numID)
					configsById[strId] = conf
				}
			}
		}
	}
	return configsById
}
