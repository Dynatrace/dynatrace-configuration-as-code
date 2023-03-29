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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/writer"
	"github.com/spf13/afero"
	"path/filepath"
	"time"
)

type WriterContext struct {
	EnvironmentUrl         string
	ProjectToWrite         project.Project
	Auth                   manifest.Auth
	EnvironmentType        manifest.EnvironmentType
	OutputFolder           string
	ForceOverwriteManifest bool
	timestampString        string
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
	m := createManifest(writerContext)

	outputFolder := writerContext.GetOutputFolderFilePath()

	log.Debug("Persisting downloaded configurations")
	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:              fs,
		OutputDir:       outputFolder,
		ManifestName:    manifestName,
		ParametersSerde: config.DefaultParameterParsers,
	}, m, []project.Project{writerContext.ProjectToWrite})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
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

func createManifest(wc WriterContext) manifest.Manifest {
	projectDefinition := manifest.ProjectDefinitionByProjectID{
		wc.ProjectToWrite.Id: {
			Name: wc.ProjectToWrite.Id,
			Path: wc.ProjectToWrite.Id,
		},
	}

	return manifest.Manifest{
		Projects: projectDefinition,
		Environments: map[string]manifest.EnvironmentDefinition{
			wc.ProjectToWrite.Id: {
				Type: wc.EnvironmentType,
				Name: wc.ProjectToWrite.Id,
				URL: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: wc.EnvironmentUrl,
				},
				Group: "default",
				Auth:  wc.Auth,
			},
		},
	}
}
