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
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/writer"
)

type WriterContext struct {
	EnvironmentUrl  manifest.URLDefinition
	ProjectToWrite  project.Project
	Auth            manifest.Auth
	OutputFolder    string
	ForceOverwrite  bool
	timestampString string
}

func (c WriterContext) GetOutputFolderFilePath() string {
	if c.OutputFolder == "" {
		return filepath.Clean(fmt.Sprintf("download_%s/", c.timestampString))
	}
	return c.OutputFolder
}

// WriteToDisk writes all projects to the disk
func WriteToDisk(fs afero.Fs, writerContext WriterContext) error {
	writerContext.timestampString = timeutils.TimeAnchor().Format("2006-01-02-150405")

	return writeToDisk(fs, writerContext)
}

func writeToDisk(fs afero.Fs, writerContext WriterContext) error {
	log.Debug("Preparing downloaded data for persisting")

	manifestFileName := getManifestFileName(fs, writerContext)
	projectFolderName := getProjectFolderName(fs, writerContext)

	projectDefinition := manifest.ProjectDefinitionByProjectID{
		writerContext.ProjectToWrite.Id: {
			Name: writerContext.ProjectToWrite.Id,
			Path: projectFolderName,
		},
	}

	manifest := manifest.Manifest{
		Projects: projectDefinition,
		SelectedEnvironments: map[string]manifest.EnvironmentDefinition{
			writerContext.ProjectToWrite.Id: {
				Name:  writerContext.ProjectToWrite.Id,
				URL:   writerContext.EnvironmentUrl,
				Group: "default",
				Auth:  writerContext.Auth,
			},
		},
	}

	outputFolder := writerContext.GetOutputFolderFilePath()

	log.Debug("Persisting downloaded configurations")
	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:              fs,
		OutputDir:       outputFolder,
		ManifestName:    manifestFileName,
		ParametersSerde: config.DefaultParameterParsers,
	}, manifest, []project.Project{writerContext.ProjectToWrite})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return fmt.Errorf("failed to persist downloaded configurations")
	}

	log.WithFields(field.F("outputFolder", outputFolder)).Info("Downloaded configurations written to '%s'", outputFolder)
	return nil
}

func getManifestFileName(fs afero.Fs, writerContext WriterContext) string {
	manifestFileName := "manifest.yaml"
	outputFolder := writerContext.GetOutputFolderFilePath()
	defaultManifestPath := filepath.Join(outputFolder, manifestFileName)
	if exists, _ := afero.Exists(fs, defaultManifestPath); !exists {
		return manifestFileName
	}

	if writerContext.ForceOverwrite {
		log.WithFields(field.F("outputFolder", outputFolder), field.F("manifestFile", "manifest.yaml")).Info("Overwriting existing manifest.yaml in download target folder.")
		return manifestFileName
	}

	manifestFileName = fmt.Sprintf("manifest_%s.yaml", writerContext.timestampString)
	log.WithFields(field.F("outputFolder", outputFolder), field.F("manifestFile", manifestFileName)).Warn("A manifest.yaml file already exists in %q, creating %q instead.", outputFolder, manifestFileName)
	return manifestFileName
}

func getProjectFolderName(fs afero.Fs, writerContext WriterContext) string {
	projectFolderName := writerContext.ProjectToWrite.Id
	outputFolder := writerContext.GetOutputFolderFilePath()
	defaultProjectFolderPath := filepath.Join(outputFolder, writerContext.ProjectToWrite.Id)
	if exists, _ := afero.Exists(fs, defaultProjectFolderPath); !exists {
		return writerContext.ProjectToWrite.Id
	}

	if writerContext.ForceOverwrite {
		log.WithFields(field.F("outputFolder", outputFolder), field.F("projectFolder", projectFolderName)).Info("Overwriting existing project folder named %q in %q.", projectFolderName, outputFolder)
		return projectFolderName
	}

	projectFolderName = fmt.Sprintf("%s_%s", writerContext.ProjectToWrite.Id, writerContext.timestampString)
	log.WithFields(field.F("outputFolder", outputFolder), field.F("projectFolder", projectFolderName)).Warn("A project folder named %q already exists in %q, creating %q instead.", writerContext.ProjectToWrite.Id, outputFolder, projectFolderName)
	return projectFolderName
}
