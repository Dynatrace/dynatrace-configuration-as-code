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
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/writer"
	"github.com/spf13/afero"
)

func writeToDisk(fs afero.Fs, downloadedConfigs project.ConfigsPerApis, projectName string) error {

	proj, projectDefinitions := createProjectData(downloadedConfigs, projectName)

	log.Debug("Writing projects to disk")
	errs := writer.WriteProjects(&writer.WriterContext{
		Fs:              fs,
		OutputDir:       projectName,
		ParametersSerde: config.DefaultParameterParsers,
	}, projectDefinitions, []project.Project{proj})

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("error writing stuff")
	}

	log.Debug("Done writing projects to disk")
	return nil
}

func createProjectData(downloadedConfigs project.ConfigsPerApis, projectName string) (project.Project, manifest.ProjectDefinitionByProjectId) {
	configsPerApiPerEn := project.ConfigsPerApisPerEnvironments{
		projectName: downloadedConfigs,
	}

	proj := project.Project{
		Id:      projectName,
		Configs: configsPerApiPerEn,
	}

	projectDefinitions := manifest.ProjectDefinitionByProjectId{
		projectName: {
			Name: projectName,
		},
	}

	return proj, projectDefinitions
}
