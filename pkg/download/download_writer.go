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
	"path/filepath"
	"time"
)

// WriteToDisk writes all projects to the disk
func WriteToDisk(fs afero.Fs, downloadedConfigs project.ConfigsPerApis, projectName, tokenEnvVarName, environmentUrl, outputFolder string) error {
	timestampString := time.Now().Format("2006-01-02-150405")

	return writeToDisk(fs, downloadedConfigs, projectName, tokenEnvVarName, environmentUrl, outputFolder, timestampString)
}

func writeToDisk(fs afero.Fs, downloadedConfigs project.ConfigsPerApis, projectName, tokenEnvVarName, environmentUrl, outputFolder, timestampString string) error {

	log.Debug("Preparing downloaded data for persisting")

	if outputFolder == "" {
		outputFolder = filepath.Clean(fmt.Sprintf("download_%s/", timestampString))
	}

	manifestName := "manifest.yaml"
	if exists, _ := afero.Exists(fs, filepath.Join(outputFolder, manifestName)); exists {
		manifestName = fmt.Sprintf("manifest_%s.yaml", timestampString)
		log.Warn("A manifest.yaml already exists in '%s', creating '%s' instead.", outputFolder, manifestName)
	}

	proj, projectDefinitions := createProjectData(downloadedConfigs, projectName)

	m := manifest.Manifest{
		Projects: projectDefinitions,
		Environments: map[string]manifest.EnvironmentDefinition{
			projectName: manifest.NewEnvironmentDefinition(projectName,
				manifest.UrlDefinition{
					Type:  manifest.ValueUrlType,
					Value: environmentUrl,
				},
				"default",
				&manifest.EnvironmentVariableToken{
					EnvironmentVariableName: tokenEnvVarName,
				}),
		},
	}

	log.Debug("Persisting downloaded configurations")
	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:              fs,
		OutputDir:       outputFolder,
		ManifestName:    manifestName,
		ParametersSerde: config.DefaultParameterParsers,
	}, m, []project.Project{proj})

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to persist downloaded configurations")
	}

	log.Info("Downloaded configurations written to '%s'", outputFolder)
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
			Path: projectName,
		},
	}

	return proj, projectDefinitions
}
