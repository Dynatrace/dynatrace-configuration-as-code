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

package deploy

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"path/filepath"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/client"
	configV2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/converter"
	configDelete "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/delete/v2"
	environmentV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	projectV1 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v1"
	projectV2 "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

func Deploy(fs afero.Fs, workingDir string, environmentsFile string,
	specificEnvironment string, specificProjects string, dryRun bool, continueOnError bool) error {
	apis := api.NewV1Apis()

	workingDir, err := filepath.Abs(workingDir)

	if err != nil {
		return fmt.Errorf("cannot transform workingDir to absolute path: %w", err)
	}

	m, projects, configLoadErrors := loadConfigs(fs, apis, environmentsFile,
		specificEnvironment, specificProjects, workingDir)

	if len(configLoadErrors) > 0 {
		util.PrintErrors(configLoadErrors)

		return fmt.Errorf("encountered errors while trying to load configs. check logs")
	}

	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, toEnvironmentNames(m.Environments))

	if errs != nil {
		util.PrintErrors(configLoadErrors)
		return fmt.Errorf("error during sort")
	}

	deploymentErrors := make(map[string][]error)
	environmentMap := m.Environments

	for envName, configs := range sortedConfigs {
		env, found := environmentMap[envName]

		if !found {
			if continueOnError {
				deploymentErrors[envName] = []error{fmt.Errorf("cannot find environment `%s`", envName)}
				continue
			} else {
				return fmt.Errorf("cannot find environment `%s`", envName)
			}
		}

		deployErrors := deployEnvironment(env, apis, configs, dryRun, continueOnError)

		if deployErrors != nil {
			if continueOnError || dryRun {
				deploymentErrors[envName] = deployErrors
				continue
			} else {
				deploymentErrors[envName] = deployErrors
				break
			}
		}
	}

	if len(deploymentErrors) > 0 {
		numberOfError := 0

		for envName, errors := range deploymentErrors {
			numberOfError = numberOfError + len(errors)

			log.Error("Environment `%s`: %d error(s)", envName, len(errors))
			util.PrintErrors(errors)
		}

		if dryRun {
			return fmt.Errorf("dry run found %d errors. check logs", numberOfError)
		} else {
			return fmt.Errorf("%d errors during deployment. check logs", numberOfError)
		}
	} else {
		if dryRun {
			log.Info("Config seems valid!")
		}
	}

	deleteErrors := deleteConfigs(fs, apis, environmentMap, workingDir, dryRun)

	if len(deleteErrors) > 0 {
		log.Error("Errors during delete:")
		util.PrintErrors(deleteErrors)

		if dryRun {
			return fmt.Errorf("dry run found %d errors. check logs", len(deleteErrors))
		} else {
			return fmt.Errorf("%d errors during deployment. check logs", len(deleteErrors))
		}
	}

	return nil
}

func toEnvironmentNames(environments map[string]manifest.EnvironmentDefinition) []string {
	result := make([]string, 0, len(environments))

	for _, env := range environments {
		result = append(result, env.Name)
	}

	return result
}

func deployEnvironment(environment manifest.EnvironmentDefinition, apis map[string]api.Api,
	sortedConfigs []configV2.Config, dryRun bool, continueOnError bool) []error {

	apiClient, err := createClient(environment, dryRun)

	if err != nil {
		return []error{err}
	}

	return deploy.DeployConfigs(apiClient, apis, sortedConfigs, deploy.DeployConfigsOptions{ContinueOnErr: continueOnError, DryRun: dryRun})
}

func createClient(environment manifest.EnvironmentDefinition, dryRun bool) (client.Client, error) {
	if dryRun {
		return &client.DummyClient{}, nil
	}

	token, err := environment.GetToken()

	if err != nil {
		return nil, err
	}

	url, err := environment.GetUrl()
	if err != nil {
		return nil, err
	}

	return client.NewDynatraceClient(url, token)
}

func loadConfigs(fs afero.Fs, apis map[string]api.Api, environmentsFile string,
	specificEnvironment string, specificProjects string, workingDir string) (manifest.Manifest, []projectV2.Project, []error) {

	environments, errors := environmentV1.LoadEnvironmentList(specificEnvironment, environmentsFile, fs)

	if len(errors) > 0 {
		return manifest.Manifest{}, nil, errors
	}

	workingDirFs := afero.NewBasePathFs(fs, workingDir)

	projects, err := projectV1.LoadProjectsToDeploy(workingDirFs, specificProjects, apis, ".")

	if err != nil {
		return manifest.Manifest{}, nil, []error{err}
	}

	return converter.Convert(converter.ConverterContext{
		Fs: workingDirFs,
	}, environments, projects)
}

func deleteConfigs(fs afero.Fs, apis api.ApiMap, environments map[string]manifest.EnvironmentDefinition,
	workingDir string, dryRun bool) []error {
	deleteFile := "delete.yaml"

	exists, err := files.DoesFileExist(fs, filepath.Join(workingDir, deleteFile))

	if err != nil {
		return []error{err}
	}

	if !exists {
		// nothing to do
		return nil
	}

	apiNames := api.GetApiNames(apis)

	entriesToDelete, errors := configDelete.LoadEntriesToDelete(fs, apiNames, workingDir, deleteFile)

	if errors != nil {
		return errors
	}

	logDeleteInfo(entriesToDelete)

	if dryRun {
		return nil
	}

	var result []error

	for _, env := range environments {
		c, err := createClient(env, false)

		if err != nil {
			result = append(result, err)
		}

		errs := configDelete.DeleteConfigs(c, apis, entriesToDelete)

		if errs != nil {
			result = append(result, errs...)
		}
	}

	return result
}

func logDeleteInfo(entriesToDelete map[string][]configDelete.DeletePointer) {
	log.Info("Trying to delete the following configs:")

	for a, entries := range entriesToDelete {
		log.Info("%s (%d):", a, len(entries))

		for _, entry := range entries {
			log.Info("\t%s", entry.ConfigId)
		}
	}
}
