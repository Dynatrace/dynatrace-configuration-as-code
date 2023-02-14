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
	config "github.com/dynatrace/dynatrace-configuration-as-code/internal/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/internal/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/util"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/util/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/writer"
	"github.com/spf13/afero"
	"path/filepath"
	"time"
)

type WriterContext struct {
	ProjectToWrite         project.Project
	TokenEnvVarName        string
	EnvironmentUrl         string
	OutputFolder           string
	ForceOverwriteManifest bool

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
	writerContext.timestampString = time.Now().Format("2006-01-02-150405")

	return writeToDisk(fs, writerContext)
}

func writeToDisk(fs afero.Fs, writerContext WriterContext) error {

	log.Debug("Preparing downloaded data for persisting")

	manifestName := getManifestFilePath(fs, writerContext)
	m := createManifest(writerContext.ProjectToWrite, writerContext.TokenEnvVarName, writerContext.EnvironmentUrl)

	outputFolder := writerContext.GetOutputFolderFilePath()

	log.Debug("Persisting downloaded configurations")
	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:              fs,
		OutputDir:       outputFolder,
		ManifestName:    manifestName,
		ParametersSerde: config.DefaultParameterParsers,
	}, m, []project.Project{writerContext.ProjectToWrite})

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to persist downloaded configurations")
	}

	log.Info("Downloaded configurations written to '%s'", outputFolder)
	return nil
}

func getManifestFilePath(fs afero.Fs, writerContext WriterContext) string {
	manifestName := "manifest.yaml"
	outputFolder := writerContext.GetOutputFolderFilePath()
	defaultManifestPath := filepath.Join(outputFolder, manifestName)
	if exists, _ := afero.Exists(fs, defaultManifestPath); !exists {
		return manifestName
	}

	if writerContext.ForceOverwriteManifest {
		log.Info("Overwriting existing manifest.yaml in download target folder.")
		return manifestName
	}

	log.Warn("A manifest.yaml already exists in '%s', creating '%s' instead.", outputFolder, manifestName)
	return fmt.Sprintf("manifest_%s.yaml", writerContext.timestampString)
}

func createManifest(proj project.Project, tokenEnvVarName string, environmentUrl string) manifest.Manifest {
	projectDefinition := manifest.ProjectDefinitionByProjectId{
		proj.Id: {
			Name: proj.Id,
			Path: proj.Id,
		},
	}

	return manifest.Manifest{
		Projects: projectDefinition,
		Environments: map[string]manifest.EnvironmentDefinition{
			proj.Id: manifest.NewEnvironmentDefinition(proj.Id,
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
}
