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

package sort

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

// ConfigsPerEnvironment returns a sorted slice of configurations for each environment. If configurations depend
// on each other, the slices will contain them in the right order to deploy one after the other.
// Depending on the configuration of featureflags.DependencyGraphBasedSort this will either use topologysort or a new graph datastructure
// based sort. To use the full graph-based implementation use graph.New instead.
func ConfigsPerEnvironment(projects []project.Project, environments []string) (sortedConfigsPerEnv project.ConfigsPerEnvironment, errs []error) {
	return graph.SortProjects(projects, environments)
}
