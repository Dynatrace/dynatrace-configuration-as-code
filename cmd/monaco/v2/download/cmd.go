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
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	configv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/downloader"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	projectv2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/writer"
	"github.com/spf13/afero"
)

const (
	DefaultGroup = "default"
	EnvName      = "download-env"
	ProjectName  = "download-project"
)

// Entrypoint for download functionality
// This class prepares the payload to be send to the download pkg by applying filters and executing the process

//GetConfigsFrom gets inputs from the CLI and creates the configuration files
func GetConfigsFrom(fs afero.Fs, workingDir string, url string,
	manifestName string, outputFolder string, filter string, tokenName string) error {

	workingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return fmt.Errorf("cannot transform workingDir to absolute path: %s", err)
	}
	environments := generateDownloadEnvironment(EnvName, url, tokenName)
	projectDefinitions, projects, errs := generateProjectDefiniton(filter, ProjectName, environments[EnvName])
	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("encountered errors while trying to download configurations. check logs")
	}
	_, errs = generateManifest(fs, workingDir, manifestName, outputFolder, environments, projectDefinitions, projects)
	if errs != nil {
		util.PrintErrors(errs)
		return fmt.Errorf("encountered errors while trying to persist configurations. check logs")
	}
	return nil
}

//generateDownloadEnvironment creates a new environment definition based on cli parameters, only a single env is generated
func generateDownloadEnvironment(envName string, url string, tokenName string) map[string]manifest.EnvironmentDefinition {
	environments := make(map[string]manifest.EnvironmentDefinition)
	env := manifest.EnvironmentDefinition{
		Name:  envName,
		Url:   url,
		Group: DefaultGroup,
		Token: &manifest.EnvironmentVariableToken{EnvironmentVariableName: tokenName},
	}
	environments[envName] = env
	return environments
}

//generateProjectDefiniton defines a new project for the download process
func generateProjectDefiniton(filter string, projectName string, env manifest.EnvironmentDefinition) (map[string]manifest.ProjectDefinition, []projectv2.Project, []error) {
	var projectList []projectv2.Project
	projectDefinitions := make(map[string]manifest.ProjectDefinition)

	projectDefinition, singleProject, errors := getSingleProject(filter, projectName, env)
	if len(errors) > 0 {
		return nil, nil, errors
	}
	projectDefinitions[projectDefinition.Name] = projectDefinition
	projectList = append(projectList, singleProject)
	return projectDefinitions, projectList, nil
}

//getSingleProject get configurations for a single project
func getSingleProject(filter string, projectName string, env manifest.EnvironmentDefinition) (manifest.ProjectDefinition, projectv2.Project, []error) {
	var errors []error
	list, err := getAPIList(filter)
	if err != nil {
		errors = append(errors, err)
		return manifest.ProjectDefinition{}, projectv2.Project{}, errors
	}
	configs, errors := downloader.DownloadConfigs(env, list, projectName)
	if errors != nil {
		return manifest.ProjectDefinition{}, projectv2.Project{}, errors
	}

	dependenciesPerEnvironment := downloader.GetDependencies()

	return manifest.ProjectDefinition{
			Name: projectName,
			Path: projectName,
		}, projectv2.Project{
			Id:           projectName,
			Configs:      configs,
			Dependencies: dependenciesPerEnvironment,
		}, nil
}

//generateManifest builds a manifest with a single environment and project
func generateManifest(fs afero.Fs, workdir string,
	manifestName string, outputFolder string, envs map[string]manifest.EnvironmentDefinition, projectDefinitions map[string]manifest.ProjectDefinition,
	projects []projectv2.Project) (manifest.Manifest, []error) {

	man := manifest.Manifest{
		Projects:     projectDefinitions,
		Environments: envs,
	}
	manifestPath := filepath.Join(workdir, outputFolder)

	errs := writer.WriteToDisk(&writer.WriterContext{
		Fs:                 fs,
		SourceManifestPath: manifestPath,
		OutputDir:          outputFolder,
		ManifestName:       manifestName,
		ParametersSerde:    configv2.DefaultParameterParsers,
	}, man, projects)
	if len(errs) > 0 {
		log.Fatal("error creating manifest file")
		return manifest.Manifest{}, errs
	}
	return man, nil
}

//AUXILIARY FUNCTIONS
//returns the list of API filter if the download specific flag is used, otherwise returns all the API's
func getAPIList(downloadSpecificAPI string) (filterAPIList map[string]api.Api, err error) {
	availableApis := api.NewApis()
	noFilterAPIListProvided := strings.TrimSpace(downloadSpecificAPI) == ""

	if noFilterAPIListProvided {
		return availableApis, nil
	}
	requestedApis := strings.Split(downloadSpecificAPI, ",")
	isErr := false
	filterAPIList = make(map[string]api.Api)
	for _, id := range requestedApis {
		cleanAPI := strings.TrimSpace(id)
		isAPI := api.IsApi(cleanAPI)
		if !isAPI {
			log.Error("Value %s is not a valid API name", cleanAPI)
			isErr = true
		} else {
			filterAPI := availableApis[cleanAPI]
			filterAPIList[cleanAPI] = filterAPI
		}
	}
	if isErr {
		return nil, fmt.Errorf("There were some errors in the API list provided")
	}

	return filterAPIList, nil
}
