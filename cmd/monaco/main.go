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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/delete"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
)

func main() {
	statusCode := Run(os.Args)
	os.Exit(statusCode)
}

func Run(args []string) int {
	return RunImpl(args, util.NewFileReader())
}

func RunImpl(args []string, fileReader util.FileReader) (statusCode int) {

	statusCode = 0

	dryRun, verbose, environments, projectNameToDeploy, path, errorList, flagError := parseInputCommand(args, fileReader)

	if flagError != nil {
		util.FailOnError(flagError, "could not parse flags")
	}

	var deploymentErrors = make(map[string]error)

	for i, err := range errorList {
		configIssue := fmt.Sprintf("environmentfile-issue-%d", i)
		deploymentErrors[configIssue] = err
	}

	err := util.SetupLogging(verbose)

	if err != nil {
		util.Log.Error("Error writing log file: %s", err.Error())
	}

	util.Log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

	apis := createApis()

	projects, err := project.LoadProjectsToDeploy(projectNameToDeploy, apis, path, fileReader)
	if err != nil {
		util.FailOnError(err, "Loading of projects failed")
	}

	util.Log.Info("Executing projects in this order: ")

	for i, project := range projects {
		util.Log.Info("\t%d: %s (%d configs)", i+1, project.GetId(), len(project.GetConfigs()))
	}

	for _, environment := range environments {
		err := execute(environment, projects, dryRun, path)
		if err != nil {
			deploymentErrors[environment.GetId()] = err
		}
	}

	for environment, err := range deploymentErrors {
		if dryRun {
			util.Log.Error("Validation of %s failed with error %s\n", environment, err)
		} else {
			util.Log.Error("Deployment to %s failed with error %s\n", environment, err)
		}
		statusCode = -1
	}

	if statusCode == 0 {
		if dryRun {
			util.Log.Info("Validation finished without errors")
		} else {
			util.Log.Info("Deployment finished without errors")
		}
		deleteConfigs(apis, environments, path, dryRun, fileReader)
	}

	return statusCode
}

func parseInputCommand(args []string, fileReader util.FileReader) (dryRun bool, verbose bool, environments map[string]environment.Environment, project string, path string, errorList []error, flagError error) {

	// define flags
	var environmentsFile string
	var specificEnvironment string

	// parse flags
	shorthand := " (shorthand)"
	dryRunUsage := "Set dry-run flag to just validate configurations instead of deploying."

	flagSet := flag.NewFlagSet("arguments", flag.ExitOnError)
	flagSet.BoolVar(&dryRun, "dry-run", false, dryRunUsage)
	flagSet.BoolVar(&dryRun, "d", false, dryRunUsage+shorthand)

	verboseUsage := "Set verbose flag to enable debug logging."
	flagSet.BoolVar(&verbose, "verbose", false, verboseUsage)
	flagSet.BoolVar(&verbose, "v", false, verboseUsage+shorthand)

	projectUsage := "Project configuration to deploy. Also deploys any dependent configuration."
	flagSet.StringVar(&project, "project", "", projectUsage)
	flagSet.StringVar(&project, "p", "", projectUsage+shorthand)

	specificEnvironmentUsage := "Specific environment (from list) to deploy to."
	flagSet.StringVar(&specificEnvironment, "specific-environment", "", specificEnvironmentUsage)
	flagSet.StringVar(&specificEnvironment, "se", "", specificEnvironmentUsage+shorthand)

	environmentsUsage := "Mandatory yaml file containing environments to deploy to."
	flagSet.StringVar(&environmentsFile, "environments", "", environmentsUsage)
	flagSet.StringVar(&environmentsFile, "e", "", environmentsUsage+shorthand)

	err := flagSet.Parse(args[1:])
	if err != nil {
		return dryRun, verbose, environments, project, path, nil, err
	}

	// Show usage if flags are invalid
	if environmentsFile == "" {
		println("Please provide environments yaml with -e/--environments!")
		flagSet.Usage()
		os.Exit(1)
	}

	environments, errorList = environment.LoadEnvironmentList(specificEnvironment, environmentsFile, fileReader)

	path = readPath(args, fileReader)

	return dryRun, verbose, environments, project, path, errorList, nil
}

func readPath(args []string, fileReader util.FileReader) string {

	// Check for path at the end:
	potentialPath := args[len(args)-1]
	if !strings.HasSuffix(potentialPath, ".yaml") {
		_, err := fileReader.ReadDir(potentialPath)
		if err == nil {
			if !strings.HasSuffix(potentialPath, string(os.PathSeparator)) {
				potentialPath += string(os.PathSeparator)
			}
			return potentialPath
		}
	}
	return ""
}

// createApis contains all api definitions.
// Some of the APIs this tool uses are 'Earlier Adopter' APIs
// and those apis are marked with comment
func createApis() map[string]api.Api {
	return api.NewApis()
}

func execute(environment environment.Environment, projects []project.Project, dryRun bool, path string) error {
	util.Log.Info("Processing environment " + environment.GetId() + "...")

	dict := make(map[string]api.DynatraceEntity)
	var nameDict = make(map[string]string)
	var name, configID string

	for _, project := range projects {

		util.Log.Info("\tProcessing project " + project.GetId() + "...")
		util.Log.Debug("\t\tDeploying configs in this order: ")
		for i, config := range project.GetConfigs() {
			util.Log.Debug("\t\t\t%d: %s", i+1, config.GetFilePath())
		}

		for _, config := range project.GetConfigs() {

			var entity api.DynatraceEntity
			var err error

			if config.IsSkipDeployment(environment) {
				util.Log.Info("\t\t\tskipping deployment of %s: %s", config.GetId(), config.GetFilePath())
				continue
			}

			name, err = config.GetObjectNameForEnvironment(environment, dict)
			if err != nil {
				return err
			}
			name = config.GetApi().GetId() + "/" + name
			configID = config.GetFullQualifiedId()
			if nameDict[name] != "" {
				return fmt.Errorf("duplicate UID '%s' found in %s and %s", name, configID, nameDict[name])
			}
			nameDict[name] = configID

			if dryRun {
				entity, err = validateConfig(project, config, dict, environment)
			} else {
				entity, err = uploadConfig(config, dict, environment)
			}

			if err != nil {
				return err
			}

			referenceId := strings.TrimPrefix(config.GetFullQualifiedId(), path)
			if entity.Name != "" {
				dict[referenceId] = entity
			}
		}
	}
	return nil
}

func validateConfig(project project.Project, config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	util.Log.Debug("\t\tValidating config " + config.GetFilePath())

	jsonString, err := config.GetConfigForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	randomId := "random-" + strconv.Itoa(rand.Int())

	// If configuration deployment skipped but has dependency, throw an error
	if config.IsSkipDeployment(environment) {
		util.Log.Info("\t\t\tskipping deployment of %s: %s", config.GetId(), config.GetFilePath())
		erronousDependencies := make([]string, 0)

		for _, requiredId := range config.GetRequiredByConfigIdList() {
			//TODO this won't work for inter project dependencies
			requiredConfig, err := project.GetConfig(requiredId)

			if err != nil {
				util.Log.Warn("Encountered known bug (cross project skipDeployment check is not working at the moment): %s", err)
				// return api.DynatraceEntity{
				// 	Id:          randomId,
				// 	Name:        randomId,
				// 	Description: randomId,
				// }, fmt.Errorf("config with id %s hasn't been found in project %s", requiredId, project.GetId())
				continue
			}

			requiredIsSkipped := requiredConfig.IsSkipDeployment(environment)

			if !requiredIsSkipped {
				erronousDependencies = append(erronousDependencies, requiredId)
			}
		}

		if len(erronousDependencies) > 0 {
			return api.DynatraceEntity{
				Id:          randomId,
				Name:        randomId,
				Description: randomId,
			}, fmt.Errorf("this config is required by %s and can't be skipped for deployment", erronousDependencies)
		}

	}

	err = validateConfigJson(jsonString, config.GetFilePath())

	return api.DynatraceEntity{
		Id:          randomId,
		Name:        randomId,
		Description: randomId,
	}, err
}

func uploadConfig(config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	util.Log.Debug("\t\tApplying config " + config.GetFilePath())

	configType := config.GetType()
	url := config.GetApi().GetUrl(environment)
	apiToken, err := environment.GetToken()
	if err != nil {
		return entity, err
	}
	jsonString, err := config.GetConfigForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	name, err := config.GetObjectNameForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	switch configType {
	case "extension":
		entity, err = rest.UploadExtension(url, name, jsonString, apiToken)
	default:
		entity, err = rest.UpsertDynatraceObject(url, name, configType, jsonString, apiToken)
	}

	if err != nil {
		err = fmt.Errorf("%s, responsible config: %s", err.Error(), config.GetFilePath())
	}
	return entity, err
}

/* validates whether the json file is correct, by using the internal validation done
 * when unmarshalling to a an object. As none of our jsons can actually be unmarshalled
 * to a string, we catch that error, but report any other error as fatal.
 */
func validateConfigJson(jsonString string, filename string) error {
	var j string
	err := json.Unmarshal([]byte(jsonString), &j)

	if err != nil && !strings.Contains(err.Error(), "cannot unmarshal") {
		return fmt.Errorf("%s is not a valid json! Error: %s", filename, err.Error())
	}

	return nil
}

// deleteConfigs deletes specified configs, if a delete.yaml file was found
func deleteConfigs(apis map[string]api.Api, environments map[string]environment.Environment, path string, dryRun bool, fileReader util.FileReader) {

	configs, err := delete.LoadConfigsToDelete(apis, path, fileReader)
	util.FailOnError(err, "deletion failed")

	if len(configs) > 0 && !dryRun {

		for name, environment := range environments {
			util.Log.Info("Deleting %d configs for environment %s...", len(configs), name)

			for _, config := range configs {
				util.Log.Debug("\tDeleting config " + config.GetId() + " (" + config.GetApi().GetId() + ")")
				_ = rest.DeleteDynatraceObject(config, environment)
			}
		}
	}
}
