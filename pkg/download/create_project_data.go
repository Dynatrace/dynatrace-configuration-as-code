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
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

func CreateProjectData(downloadedConfigs project.ConfigsPerType, projectName string) project.Project {
	configsPerTypePerEnv := project.ConfigsPerTypePerEnvironments{
		projectName: downloadedConfigs,
	}

	proj := project.Project{
		Id:      projectName,
		Configs: configsPerTypePerEnv,
	}

	return proj
}
