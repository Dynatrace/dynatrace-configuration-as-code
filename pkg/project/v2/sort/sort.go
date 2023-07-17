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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort/topologysort"
)

func SortParameters(group string, environment string, conf coordinate.Coordinate, parameters config.Parameters) ([]parameter.NamedParameter, []error) {
	return topologysort.SortParameters(group, environment, conf, parameters)
}

func GetSortedConfigsForEnvironments(projects []project.Project, environments []string) (sortedConfigsPerEnv project.ConfigsPerEnvironment, errs []error) {
	return topologysort.SortProjects(projects, environments)
}
