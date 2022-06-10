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
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/delete"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/spf13/afero"
)

type Deploy struct {
	fs                   afero.Fs
	workingDir           string
	environmentsFilePath string
	loadEnvironmentList  func(specificEnvironment string, environmentsFilePath string, fs afero.Fs) (environments map[string]environment.Environment, errorList []error)
	loadProjectsToDeploy func(fs afero.Fs, specificProjectToDeploy string, apis map[string]api.Api, path string) (projectsToDeploy []project.Project, err error)
	loadConfigsToDelete  func(fs afero.Fs, apis map[string]api.Api, path string) (configs []config.Config, err error)
	failOnError          func(err error, msg string)
	apis                 map[string]api.Api
	newDynatraceClient   func(environmentUrl string, token string) (rest.DynatraceClient, error)
}

type DeployIface interface {
	Deploy(specificEnvironment string, proj string, continueOnError bool) error
	DeployDryRun(specificEnvironment string, proj string, continueOnError bool) error
	Delete(specificEnvironment string, proj string) error
	DeleteDryRun(specificEnvironment string, proj string) error
	RunAll(specificEnvironment string, proj string, isDryRun bool, continueOnError bool) error
}

func returnFromDeploymentWithError(isDryRun bool, continueOnError bool, deploymentErrors map[string][]error) error {
	util.Log.Info("Deployment summary:")

	if isDryRun {
		for environment, errors := range deploymentErrors {
			util.Log.Error("Validation of %s failed. Found %d error(s)\n", environment, len(errors))
			util.PrintErrors(errors)
		}

		return fmt.Errorf("errors during validation! Check log")
	} else if continueOnError {
		for environment, errors := range deploymentErrors {
			util.Log.Error("Deployment to %s finished with %d error(s):\n", environment, len(errors))
			util.PrintErrors(errors)
		}

		return fmt.Errorf("errors during deployment! Check log")
	} else {
		for environment, errors := range deploymentErrors {
			util.Log.Error("Deployment to %s failed with error!\n", environment)
			util.PrintErrors(errors)
		}

		return fmt.Errorf("errors during deployment! Check log")
	}
}

func returnFromDeployment(isDryRun bool) error {
	util.Log.Info("Deployment summary:")

	if isDryRun {
		util.Log.Info("Validation finished without errors")
	} else {
		util.Log.Info("Deployment finished without errors")
	}

	return nil
}

func NewHandler(
	workingDir string,
	fs afero.Fs,
	environmentsFilePath string,
) (*Deploy, error) {
	// TBD: Check whether workingDir, environmentsFilePath exist...

	deploy := &Deploy{
		fs:                   fs,
		workingDir:           filepath.Clean(workingDir),
		environmentsFilePath: environmentsFilePath,
		loadEnvironmentList:  environment.LoadEnvironmentList,
		loadProjectsToDeploy: project.LoadProjectsToDeploy,
		loadConfigsToDelete:  delete.LoadConfigsToDelete,
		failOnError:          util.FailOnError,
		apis:                 api.NewApis(),
		newDynatraceClient:   rest.NewDynatraceClient,
	}

	return deploy, nil
}

func (d *Deploy) deploy(
	specificEnvironment string,
	proj string,
	isDryRun bool,
	continueOnError bool,
) error {
	environments, errors := d.loadEnvironmentList(specificEnvironment, d.environmentsFilePath, d.fs)

	var deploymentErrors = make(map[string][]error)

	for i, err := range errors {
		configIssue := fmt.Sprintf("environmentfile-issue-%d", i)
		deploymentErrors[configIssue] = append(deploymentErrors[configIssue], err)
	}

	projects, err := d.loadProjectsToDeploy(d.fs, proj, d.apis, d.workingDir)
	if err != nil {
		d.failOnError(err, "Loading of projects failed")
	}

	util.Log.Info("Executing projects in this order: ")

	for i, project := range projects {
		util.Log.Info("\t%d: %s (%d configs)", i+1, project.GetId(), len(project.GetConfigs()))
	}

	for _, environment := range environments {
		errors := d.execute(environment, projects, isDryRun, continueOnError)
		if len(errors) > 0 {
			deploymentErrors[environment.GetId()] = errors
		}
	}

	isErrored := len(deploymentErrors) > 0
	if isErrored {
		return returnFromDeploymentWithError(isDryRun, continueOnError, deploymentErrors)
	} else {
		return returnFromDeployment(isDryRun)
	}
}

func (d *Deploy) Deploy(
	specificEnvironment string,
	proj string,
	continueOnError bool,
) error {
	isDryRun := false

	return d.deploy(specificEnvironment, proj, isDryRun, continueOnError)
}

func (d *Deploy) DeployDryRun(
	specificEnvironment string,
	proj string,
	continueOnError bool,
) error {
	isDryRun := true

	return d.deploy(specificEnvironment, proj, isDryRun, continueOnError)
}

func (d *Deploy) Delete(
	specificEnvironment string,
	proj string,
) error {
	isDryRun := false

	environments, _ := d.loadEnvironmentList(specificEnvironment, d.environmentsFilePath, d.fs)
	return d.deleteConfigs(environments, isDryRun)
}

func (d *Deploy) DeleteDryRun(
	specificEnvironment string,
	proj string,
) error {
	isDryRun := true

	environments, _ := d.loadEnvironmentList(specificEnvironment, d.environmentsFilePath, d.fs)
	return d.deleteConfigs(environments, isDryRun)
}

func (d *Deploy) RunAll(
	specificEnvironment string,
	proj string,
	isDryRun bool,
	continueOnError bool,
) error {
	err := d.deploy(specificEnvironment, proj, isDryRun, continueOnError)
	// do not execute delete if there are problems with deployment
	if err != nil {
		return err
	}

	environments, _ := d.loadEnvironmentList(specificEnvironment, d.environmentsFilePath, d.fs)
	err = d.deleteConfigs(environments, isDryRun)
	if err != nil {
		return err
	}

	return nil
}

func validateOrUpload(
	isDryRun bool,
	project project.Project,
	client rest.DynatraceClient,
	config config.Config,
	dict map[string]api.DynatraceEntity,
	environment environment.Environment,
) (entity api.DynatraceEntity, err error) {
	if isDryRun {
		return validateConfig(project, config, dict, environment)
	} else {
		return uploadConfig(project, client, config, dict, environment)
	}
}

func logProjectDeployOrder(projectId string, configs []config.Config) {
	util.Log.Info("\tProcessing project %s...", projectId)
	util.Log.Debug("\t\tDeploying configs in this order: ")
	for i, config := range configs {
		util.Log.Debug("\t\t\t%d: %s", i+1, config.GetFilePath())
	}
}

func executeConfig(
	isDryRun bool,
	continueOnError bool,
	environment environment.Environment,
	project project.Project,
	workingDir string,
	config config.Config,
	dict map[string]api.DynatraceEntity,
	nameDict map[string]string,
	client rest.DynatraceClient,
	errors *[]error,
) error {
	if config.IsSkipDeployment(environment) {
		util.Log.Info("\t\t\tskipping deployment of %s: %s", config.GetId(), config.GetFilePath())
		return nil
	}

	isNonUniqueNameApi := config.GetApi().IsNonUniqueNameApi()
	if !isNonUniqueNameApi {
		name, err := config.GetObjectNameForEnvironment(environment, dict)
		if err != nil {
			return err
		}
		name = config.GetApi().GetId() + "/" + name
		configID := config.GetFullQualifiedId()
		if nameDict[name] != "" {
			return fmt.Errorf("duplicate UID '%s' found in %s and %s", name, configID, nameDict[name])
		}
		nameDict[name] = configID
	}

	entity, err := validateOrUpload(isDryRun, project, client, config, dict, environment)

	if err != nil {
		// by default stop deployment on error
		if continueOnError || isDryRun {
			*errors = append(*errors, err)
			// Log error here in addition to deployment summary
			// Useful to debug using verbose
			util.Log.Error("\t\t\tFailed %s", err)
		} else {
			return err
		}
	}

	referenceId := strings.TrimPrefix(config.GetFullQualifiedId(), workingDir+"/")

	if entity.Name != "" {
		dict[referenceId] = entity
	}

	return nil
}

func (d *Deploy) execute(environment environment.Environment, projects []project.Project, dryRun bool, continueOnError bool) []error {
	util.Log.Info("Processing environment " + environment.GetId() + "...")

	var client rest.DynatraceClient
	if !dryRun {
		apiToken, err := environment.GetToken()
		if err != nil {
			return []error{err}
		}

		client, err = d.newDynatraceClient(environment.GetEnvironmentUrl(), apiToken)
		if err != nil {
			return []error{err}
		}
	}

	dict := make(map[string]api.DynatraceEntity)
	nameDict := make(map[string]string)
	errors := []error{}

	for _, project := range projects {
		configs := project.GetConfigs()
		projectId := project.GetId()

		logProjectDeployOrder(projectId, configs)

		for _, config := range configs {
			err := executeConfig(
				dryRun,
				continueOnError,
				environment,
				project,
				d.workingDir,
				config,
				dict,
				nameDict,
				client,
				&errors,
			)
			if err != nil {
				return append(errors, err)
			}
		}
	}

	return errors
}

func validateConfig(project project.Project, config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	util.Log.Debug("\t\tValidating config " + config.GetFilePath())

	_, err = config.GetConfigForEnvironment(environment, dict)

	if err != nil {
		return entity, err
	}

	randomId := "random-" + strconv.Itoa(rand.Int())

	return api.DynatraceEntity{
		Id:          randomId,
		Name:        randomId,
		Description: randomId,
	}, err
}

func uploadConfig(project project.Project, client rest.DynatraceClient, config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	name, err := config.GetObjectNameForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	util.Log.Debug("\t\tApplying config `%s` using %s", name, config.GetFilePath())

	uploadMap, err := config.GetConfigForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	isNonUniqueNameApi := config.GetApi().IsNonUniqueNameApi()

	if isNonUniqueNameApi {
		configId := config.GetId()

		entityId, err := project.GenerateConfigUuid(configId)
		if err != nil {
			return entity, err
		}

		entity, err = client.UpsertByEntityId(config.GetApi(), entityId, name, uploadMap)
		if err != nil {
			err = fmt.Errorf("%w, responsible config: %s", err, config.GetFilePath())
		}
		return entity, err
	} else {
		entity, err = client.UpsertByName(config.GetApi(), name, uploadMap)
		if err != nil {
			err = fmt.Errorf("%w, responsible config: %s", err, config.GetFilePath())
		}
		return entity, err
	}
}

// deleteConfigs deletes specified configs, if a delete.yaml file was found
func (d *Deploy) deleteConfigs(environments map[string]environment.Environment, dryRun bool) error {
	configs, err := d.loadConfigsToDelete(d.fs, d.apis, d.workingDir)
	util.FailOnError(err, "deletion failed")

	if len(configs) > 0 && !dryRun {

		for name, environment := range environments {
			util.Log.Info("Deleting %d configs for environment %s...", len(configs), name)

			apiToken, err := environment.GetToken()
			if err != nil {
				return err
			}

			client, err := d.newDynatraceClient(environment.GetEnvironmentUrl(), apiToken)
			if err != nil {
				return err
			}

			for _, config := range configs {
				configId := config.GetId()
				configApi := config.GetApi()
				configApiId := configApi.GetId()
				// isNonUniqueNameApi := configApi.IsNonUniqueNameApi()

				util.Log.Debug("\tDeleting config " + configId + " (" + configApiId + ")")

				err = client.DeleteByName(configApi, configId)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
