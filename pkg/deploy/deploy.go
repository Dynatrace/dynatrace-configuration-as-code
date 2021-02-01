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
)

func Deploy(workingDir string, fileReader util.FileReader, environmentsFile string,
	specificEnvironment string, proj string, dryRun bool, verbose bool) error {
	environments, errors := environment.LoadEnvironmentList(specificEnvironment, environmentsFile, fileReader)

	workingDir = filepath.Clean(workingDir)

	var deploymentErrors = make(map[string]error)

	for i, err := range errors {
		configIssue := fmt.Sprintf("environmentfile-issue-%d", i)
		deploymentErrors[configIssue] = err
	}

	apis := api.NewApis()

	projects, err := project.LoadProjectsToDeploy(proj, apis, workingDir, fileReader)
	if err != nil {
		util.FailOnError(err, "Loading of projects failed")
	}

	util.Log.Info("Executing projects in this order: ")

	for i, project := range projects {
		util.Log.Info("\t%d: %s (%d configs)", i+1, project.GetId(), len(project.GetConfigs()))
	}

	for _, environment := range environments {
		err := execute(environment, projects, dryRun, workingDir)
		if err != nil {
			deploymentErrors[environment.GetId()] = err
		}
	}

	for environment, err := range deploymentErrors {
		if dryRun {
			util.Log.Error("Validation of %s failed with error %s\n", environment, err)
			util.PrettyPrintError(err)
		} else {
			util.Log.Error("Deployment to %s failed with error %s\n", environment, err)
		}

		return fmt.Errorf("Errors during deployment! Check log!")
	}

	if dryRun {
		util.Log.Info("Validation finished without errors")
	} else {
		util.Log.Info("Deployment finished without errors")
	}

	deleteConfigs(apis, environments, workingDir, dryRun, fileReader)

	return nil
}

func execute(environment environment.Environment, projects []project.Project, dryRun bool, path string) error {
	util.Log.Info("Processing environment " + environment.GetId() + "...")

	var client rest.DynatraceClient
	if !dryRun {
		apiToken, err := environment.GetToken()
		if err != nil {
			return err
		}

		client, err = rest.NewDynatraceClient(environment.GetEnvironmentUrl(), apiToken)
		if err != nil {
			return err
		}
	}

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
				entity, err = uploadConfig(client, config, dict, environment)
			}

			if err != nil {
				return err
			}

			referenceId := strings.TrimPrefix(config.GetFullQualifiedId(), path+"/")

			if entity.Name != "" {
				dict[referenceId] = entity
			}
		}
	}
	return nil
}

func validateConfig(project project.Project, config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	util.Log.Debug("\t\tValidating config " + config.GetFilePath())

	_, err = config.GetConfigForEnvironment(environment, dict)

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

	return api.DynatraceEntity{
		Id:          randomId,
		Name:        randomId,
		Description: randomId,
	}, err
}

func uploadConfig(client rest.DynatraceClient, config config.Config, dict map[string]api.DynatraceEntity, environment environment.Environment) (entity api.DynatraceEntity, err error) {
	util.Log.Debug("\t\tApplying config " + config.GetFilePath())

	uploadMap, err := config.GetConfigForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	name, err := config.GetObjectNameForEnvironment(environment, dict)
	if err != nil {
		return entity, err
	}

	entity, err = client.UpsertByName(config.GetApi(), name, uploadMap)

	if err != nil {
		err = fmt.Errorf("%s, responsible config: %s", err.Error(), config.GetFilePath())
	}
	return entity, err
}

// deleteConfigs deletes specified configs, if a delete.yaml file was found
func deleteConfigs(apis map[string]api.Api, environments map[string]environment.Environment, path string, dryRun bool, fileReader util.FileReader) error {
	configs, err := delete.LoadConfigsToDelete(apis, path, fileReader)
	util.FailOnError(err, "deletion failed")

	if len(configs) > 0 && !dryRun {

		for name, environment := range environments {
			util.Log.Info("Deleting %d configs for environment %s...", len(configs), name)

			apiToken, err := environment.GetToken()
			if err != nil {
				return err
			}

			client, err := rest.NewDynatraceClient(environment.GetEnvironmentUrl(), apiToken)
			if err != nil {
				return err
			}

			for _, config := range configs {
				util.Log.Debug("\tDeleting config " + config.GetId() + " (" + config.GetApi().GetId() + ")")

				err = client.DeleteByName(config.GetApi(), config.GetId())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
