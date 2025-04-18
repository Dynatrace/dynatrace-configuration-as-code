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

package graph

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

// SortProjects returns a sorted slice of configurations for each environment. If configurations depend
// on each other, the slices will contain them in the right order to deploy one after the other.
// Internally it builds dependency graphs and uses these to sort and return the basic sorted configs per environment map.
func SortProjects(projects []project.Project, environments []string) (project.ConfigsPerEnvironment, []error) {
	cfgsPerEnv := make(map[string][]config.Config)
	var errs SortingErrors

	graphs := New(projects, environments)

	for _, environment := range environments {
		sortedCfgs, err := graphs.SortConfigs(environment)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cfgsPerEnv[environment] = sortedCfgs
	}

	if len(errs) > 0 {
		return map[string][]config.Config{}, errs
	}
	return cfgsPerEnv, nil
}
