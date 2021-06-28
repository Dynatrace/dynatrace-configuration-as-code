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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	configError "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	deploy "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/deploy/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2/topologysort"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/client"
	"github.com/spf13/afero"
)

func Deploy(fs afero.Fs, deploymentManifestPath string, specificEnvironment string,
	specificProject string, dryRun, continueOnError bool) error {

	deploymentManifestPath = filepath.Clean(deploymentManifestPath)
	deploymentManifestPath, err := filepath.Abs(deploymentManifestPath)

	if err != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %s", deploymentManifestPath, err)
	}

	manifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: deploymentManifestPath,
	})

	if errs != nil {
		// TODO add grouping and print proper error repot
		util.PrintErrors(errs)
		return errors.New("error while loading environments")
	}

	environments := manifest.GetEnvironmentsAsSlice()

	if specificEnvironment != "" {
		filtered, err := filterEnvironmentByName(environments, specificEnvironment)

		if err != nil {
			return err
		}

		environments = filtered
	}

	environmentMap := toEnvironmentMap(environments)
	environmentNames := toEnvironmentNames(environments)
	workingDir := filepath.Dir(deploymentManifestPath)

	apis := api.NewApis()

	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		Apis:            getApiNames(apis),
		WorkingDir:      workingDir,
		Manifest:        manifest,
		ParametersSerde: config.DefaultParameterParsers,
	})

	if errs != nil {
		// TODO add grouping and print proper error repot
		util.PrintErrors(errs)
		return errors.New("error while loading projects")
	}

	if specificProject != "" {
		filtered, err := filterProjectIdsByName(projects, []string{specificProject})

		if err != nil {
			return err
		}

		projectsWithDependencies, err := loadProjectsWithDependencies(projects, filtered, environmentNames)

		if err != nil {
			return err
		}

		projects = projectsWithDependencies
	}

	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, environmentNames)

	if errs != nil {
		// TODO add grouping and print proper error repot
		util.PrintErrors(errs)
		return errors.New("error during sort")
	}

	var deploymentErrors []error

	for envName, configs := range sortedConfigs {
		util.Log.Info("Processing environment `%s`...", envName)
		env, found := environmentMap[envName]

		if !found {
			if continueOnError {
				deploymentErrors = append(deploymentErrors, fmt.Errorf("cannot find environment `%s`", envName))
				continue
			} else {
				return fmt.Errorf("cannot find environment `%s`", envName)
			}
		}

		client, err := getClient(env, dryRun)

		if err != nil {
			if continueOnError {
				deploymentErrors = append(deploymentErrors, err)
				continue
			} else {
				return err
			}
		}

		errors := deploy.DeployConfigs(client, apis, configs, continueOnError, dryRun)

		deploymentErrors = append(deploymentErrors, errors...)
	}

	if deploymentErrors != nil {
		printErrorReport(deploymentErrors)

		return errors.New("errors during deploy")
	} else {
		util.Log.Info("Deployment finished without errors")
	}

	return nil
}

func printErrorReport(deploymentErrors []error) {
	var configErrors []configError.ConfigError
	var generalErrors []error

	for _, err := range deploymentErrors {
		switch e := err.(type) {
		case configError.ConfigError:
			configErrors = append(configErrors, e)
		default:
			generalErrors = append(generalErrors, e)
		}
	}

	if len(generalErrors) > 0 {
		util.Log.Error("=== General Errors ===")
		for _, err := range generalErrors {
			util.Log.Error(util.ErrorString(err))
		}
	}

	groupedConfigErrors := groupConfigErrors(configErrors)

	for project, apiErrors := range groupedConfigErrors {
		util.Log.Error("==== Project `%s`\n", project)

		for api, configErrors := range apiErrors {
			util.Log.Error("===== Api `%s`\n", api)

			for config, errs := range configErrors {
				util.Log.Error("====== Config `%s`\n", config)

				var generalConfigErrors []configError.ConfigError
				var detailedConfigErrors []configError.DetailedConfigError

				for _, err := range errs {
					switch e := err.(type) {
					case configError.DetailedConfigError:
						detailedConfigErrors = append(detailedConfigErrors, e)
					default:
						generalConfigErrors = append(generalConfigErrors, e)
					}
				}

				groupErrors := groupEnvironmentConfigErrors(detailedConfigErrors)

				for _, err := range generalConfigErrors {
					util.Log.Error(util.ErrorString(err))
				}

				for group, environmentErrors := range groupErrors {
					util.Log.Error("======= Group `%s`\n", group)

					for env, errs := range environmentErrors {
						util.Log.Error("======== Env `%s`\n", env)

						for _, err := range errs {
							util.Log.Error(util.ErrorString(err))
						}
					}
				}
			}
		}
	}
}

type ProjectErrors map[string]ApiErrors
type ApiErrors map[string]ConfigErrors
type ConfigErrors map[string][]configError.ConfigError

func groupConfigErrors(errors []configError.ConfigError) ProjectErrors {
	projectErrors := make(ProjectErrors)

	for _, err := range errors {
		coord := err.Coordinates()

		apiErrors := projectErrors[coord.Project]

		if apiErrors == nil {
			apiErrors = make(ApiErrors)
			apiErrors[coord.Api] = make(ConfigErrors)
			projectErrors[coord.Project] = apiErrors
		}

		configErrors := apiErrors[coord.Api]

		if configErrors == nil {
			configErrors = make(ConfigErrors)
			apiErrors[coord.Api] = configErrors
		}

		configErrors[coord.Config] = append(configErrors[coord.Config], err)
	}

	return projectErrors
}

type GroupErrors map[string]EnvironmentErrors
type EnvironmentErrors map[string][]configError.DetailedConfigError

func groupEnvironmentConfigErrors(errors []configError.DetailedConfigError) GroupErrors {
	groupErrors := make(GroupErrors)

	for _, err := range errors {
		locationDetails := err.LocationDetails()

		envErrors := groupErrors[locationDetails.Group]

		if envErrors == nil {
			envErrors = make(EnvironmentErrors)
			groupErrors[locationDetails.Group] = envErrors
		}

		envErrors[locationDetails.Environment] = append(envErrors[locationDetails.Environment], err)
	}

	return groupErrors
}

func toEnvironmentNames(environments []manifest.EnvironmentDefinition) []string {
	result := make([]string, 0, len(environments))

	for _, env := range environments {
		result = append(result, env.Name)
	}

	return result
}

func filterProjectIdsByName(projects []project.Project, names []string) ([]string, error) {
	var result []string

	foundProjects := map[string]struct{}{}

	for _, p := range projects {
		if containsName(names, p.Id) {
			foundProjects[p.Id] = struct{}{}
			result = append(result, p.Id)
		}
	}

	var notFoundProjects []string

	for _, name := range names {
		if _, found := foundProjects[name]; !found {
			notFoundProjects = append(notFoundProjects, name)
		}
	}

	if notFoundProjects != nil {
		return nil, fmt.Errorf("no project with names `%s` found", strings.Join(names, ", "))
	}

	return result, nil
}

func loadProjectsWithDependencies(projects []project.Project, projectIdsToLoad []string, environments []string) ([]project.Project, error) {
	lookupMap := toProjectMap(projects)
	alreadyChecked := map[string]struct{}{}
	toCheck := append(make([]string, 0, len(projectIdsToLoad)), projectIdsToLoad...)

	var result []project.Project
	var unknownProjects []string

	for len(toCheck) > 0 {
		current := toCheck[0]
		toCheck = toCheck[1:]

		if _, found := alreadyChecked[current]; found {
			continue
		}

		if project, found := lookupMap[current]; found {
			alreadyChecked[current] = struct{}{}
			result = append(result, project)

			// we need to load only the dependencies of environments we are going to deploy
			for _, env := range environments {
				toCheck = append(toCheck, project.Dependencies[env]...)
			}
		} else {
			unknownProjects = append(unknownProjects, current)
		}
	}

	if unknownProjects != nil {
		return nil, fmt.Errorf("error while gathering dependencies. no projects with name `%s` found", unknownProjects)
	}

	return result, nil
}

func toProjectMap(projects []project.Project) map[string]project.Project {
	result := make(map[string]project.Project)

	for _, p := range projects {
		result[p.Id] = p
	}

	return result
}

func containsName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}

	return false
}

func toEnvironmentMap(environments []manifest.EnvironmentDefinition) map[string]manifest.EnvironmentDefinition {
	result := make(map[string]manifest.EnvironmentDefinition)

	for _, env := range environments {
		result[env.Name] = env
	}

	return result
}

func getClient(environment manifest.EnvironmentDefinition, dryRun bool) (rest.DynatraceClient, error) {
	if dryRun {
		return &client.DummyClient{
			Entries: map[api.Api][]client.DataEntry{},
		}, nil
	} else {
		token, err := environment.GetToken()

		if err != nil {
			return nil, err
		}

		return rest.NewDynatraceClient(environment.Url, token)
	}
}

func getApiNames(apis map[string]api.Api) []string {
	result := make([]string, 0, len(apis))

	for api := range apis {
		result = append(result, api)
	}

	return result
}

func filterEnvironmentByName(environments []manifest.EnvironmentDefinition, name string) ([]manifest.EnvironmentDefinition, error) {
	var result []manifest.EnvironmentDefinition

	for _, env := range environments {
		if env.Name == name {
			result = append(result, env)
		}
	}

	if result != nil {
		return result, nil
	}

	return nil, fmt.Errorf("no environment with name `%s` found", name)
}
