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

package download

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"net/url"
	"path"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"github.com/spf13/afero"
)

//go:generate mockgen -source=download.go -destination=download_mock.go -package=download -write_package_comment=false Command

// Command is used to test the CLi commands properly without executing the actual monaco download.
//
// The actual implementations are in the [DefaultCommand] struct.
type Command interface {
	DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions manifestDownloadOptions) error
	DownloadConfigs(fs afero.Fs, cmdOptions directDownloadOptions) error
	DownloadEntitiesBasedOnManifest(fs afero.Fs, cmdOptions entitiesManifestDownloadOptions) error
	DownloadEntities(fs afero.Fs, cmdOptions entitiesDirectDownloadOptions) error
}

// DefaultCommand is used to implement the [Command] interface.
type DefaultCommand struct{}

// make sure DefaultCommand implements the Command interface
var (
	_ Command = (*DefaultCommand)(nil)
)

type downloadCommandOptionsShared struct {
	projectName    string
	outputFolder   string
	forceOverwrite bool
}

type downloadOptionsShared struct {
	environmentUrl          string
	environmentType         manifest.EnvironmentType
	auth                    manifest.Auth
	outputFolder            string
	projectName             string
	forceOverwriteManifest  bool
	concurrentDownloadLimit int
}

func writeConfigs(downloadedConfigs project.ConfigsPerType, opts downloadOptionsShared, fs afero.Fs) error {
	proj := download.CreateProjectData(downloadedConfigs, opts.projectName)

	downloadWriterContext := download.WriterContext{
		EnvironmentUrl:         opts.environmentUrl,
		ProjectToWrite:         proj,
		Auth:                   opts.auth,
		EnvironmentType:        opts.environmentType,
		OutputFolder:           opts.outputFolder,
		ForceOverwriteManifest: opts.forceOverwriteManifest,
	}
	err := download.WriteToDisk(fs, downloadWriterContext)
	if err != nil {
		return err
	}

	log.Info("Searching for circular dependencies")
	if depErr := reportForCircularDependencies(proj); depErr != nil {
		log.Warn("Download finished with problems: %s", depErr)
	} else {
		log.Info("No circular dependencies found")
	}

	log.Info("Finished download")
	return nil
}

func reportForCircularDependencies(p project.Project) error {
	_, errs := topologysort.GetSortedConfigsForEnvironments([]project.Project{p}, []string{p.Id})
	if len(errs) != 0 {
		errutils.PrintWarnings(errs)
		return fmt.Errorf("there are circular dependencies between %d configurations that need to be resolved manually", len(errs))
	}
	return nil
}

func sumConfigs(configs project.ConfigsPerType) int {
	sum := 0

	for _, v := range configs {
		sum += len(v)
	}

	return sum
}

// validateParameters checks that all necessary variables have been set.
func validateParameters(envVarName, environmentUrl, projectName, token string) []error {
	errors := make([]error, 0)

	if envVarName == "" {
		errors = append(errors, fmt.Errorf("token not specified"))
	} else if token == "" {
		errors = append(errors, fmt.Errorf("the content of token '%v' is not set", envVarName))
	}

	if environmentUrl == "" {
		errors = append(errors, fmt.Errorf("url not specified"))
	}

	if _, err := url.Parse(environmentUrl); err != nil {
		errors = append(errors, fmt.Errorf("url is invalid: %w", err))
	}

	if projectName == "" {
		errors = append(errors, fmt.Errorf("project not specified"))
	}

	return errors
}

func preDownloadValidations(fs afero.Fs, opts downloadOptionsShared) error {

	errs := validateOutputFolder(fs, opts.outputFolder, opts.projectName)
	if len(errs) > 0 {
		return errutils.PrintAndFormatErrors(errs, "output folder is invalid")
	}

	return nil
}

func validateOutputFolder(fs afero.Fs, outputFolder, project string) []error {
	errors := make([]error, 0)

	errors = append(errors, validateFolder(fs, outputFolder)...)
	if len(errors) > 0 {
		return errors
	}
	errors = append(errors, validateFolder(fs, path.Join(outputFolder, project))...)
	return errors

}

func validateFolder(fs afero.Fs, path string) []error {
	errors := make([]error, 0)
	exists, err := afero.Exists(fs, path)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to check if output folder '%s' exists: %w", path, err))
	}
	if exists {
		isDir, err := afero.IsDir(fs, path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to check if output folder '%s' is a folder: %w", path, err))
		}
		if !isDir {
			errors = append(errors, fmt.Errorf("unable to write to '%s': file exists and is not a directory", path))
		}
	}

	return errors
}
