// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package convert

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"path"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	configv2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/converter"
	environmentv1 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	projectv1 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v1"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/writer"
	"github.com/spf13/afero"
)

func convert(fs afero.Fs, workingDir string, environmentsFile string, outputFolder string,
	manifestName string) error {
	apis := api.NewV1Apis()

	log.Info("Converting configurations from '%s' ...", workingDir)
	man, projs, configLoadErrors := loadConfigs(fs, workingDir, apis, environmentsFile)

	if len(configLoadErrors) > 0 {
		errutils.PrintErrors(configLoadErrors)

		return fmt.Errorf("encountered errors while trying to load configs. check logs")
	}

	if len(projs) == 0 {
		log.Error("Please ensure that you run the command with a directory that holds monaco projects to be converted.")
		return fmt.Errorf("no projects to convert found in %q", workingDir)
	}

	manifestPath := filepath.Join(outputFolder, manifestName)

	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:                 fs,
		SourceManifestPath: manifestPath,
		OutputDir:          outputFolder,
		ManifestName:       manifestName,
		ParametersSerde:    configv2.DefaultParameterParsers,
	}, man, projs)

	if len(errs) > 0 {
		log.Error("Encountered %d errors while converting %s:", len(errs), workingDir)
		errutils.PrintErrors(errs)

		return fmt.Errorf("encountered errors while converting configs. check logs")
	}

	err := copyDeleteFileIfPresent(fs, workingDir, outputFolder)
	if err != nil {
		return fmt.Errorf("failed to copy delete.yaml from %s to %s: %w", workingDir, outputFolder, err)
	}

	log.Info("Successfully converted configurations to v2 format, stored in '%s'", outputFolder)
	return nil
}

func loadConfigs(fs afero.Fs, workingDir string, apis map[string]api.Api,
	environmentsFile string) (manifest.Manifest, []projectv2.Project, []error) {

	environments, errors := environmentv1.LoadEnvironmentsWithoutTemplating(environmentsFile, fs)

	if len(errors) > 0 {
		return manifest.Manifest{}, nil, errors
	}

	// only allow access to files inside the working dir
	var workingDirFs afero.Fs

	if workingDir == "." {
		workingDirFs = fs
	} else {
		workingDirFs = afero.NewBasePathFs(fs, workingDir)
	}

	projects, err := projectv1.LoadProjectsToConvert(workingDirFs, apis, ".")

	projects = removeEmptyProjects(projects)

	if err != nil {
		return manifest.Manifest{}, nil, []error{err}
	}

	return converter.Convert(converter.ConverterContext{
		Fs: workingDirFs,
	}, environments, projects)
}

func removeEmptyProjects(projects []projectv1.Project) []projectv1.Project {
	filteredProjects := make([]projectv1.Project, 0, len(projects))

	for _, project := range projects {

		numberConfigs := len(project.GetConfigs())

		if numberConfigs == 0 {
			log.Debug("Skipping project '%v' as it contains no configs.", project.GetId())
		} else {
			filteredProjects = append(filteredProjects, project)
		}
	}

	return filteredProjects
}

func copyDeleteFileIfPresent(fs afero.Fs, workingDir, outputFolder string) error {

	currentDeleteFile := path.Join(workingDir, "delete.yaml")

	deleteFileExists, err := afero.Exists(fs, currentDeleteFile)
	if err != nil {
		return err
	}

	if !deleteFileExists {
		return nil
	}

	f, err := afero.ReadFile(fs, currentDeleteFile)
	if err != nil {
		return err
	}
	err = afero.WriteFile(fs, path.Join(outputFolder, "delete.yaml"), f, 0644)
	if err != nil {
		return err
	}

	return nil
}
