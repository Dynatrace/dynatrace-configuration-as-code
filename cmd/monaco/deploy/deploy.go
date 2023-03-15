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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/deploy"
	"path/filepath"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	configError "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2/topologysort"
	"github.com/spf13/afero"
)

func deployConfigs(fs afero.Fs, manifestPath string, environmentGroups []string, specificEnvironments []string, specificProjects []string, continueOnErr bool, dryRun bool) error {
	absManifestPath, err := absPath(manifestPath)
	if err != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %w", manifestPath, err)
	}
	loadedManifest, err := loadManifest(fs, absManifestPath, environmentGroups, specificEnvironments)
	if err != nil {
		return err
	}

	err = verifyClusterGen(loadedManifest.Environments, dryRun)
	if err != nil {
		return err
	}

	loadedProjects, err := loadProjects(fs, absManifestPath, loadedManifest)
	if err != nil {
		return err
	}

	filteredProjects, err := filterProjects(loadedProjects, specificProjects, loadedManifest.Environments.Names())
	if err != nil {
		return fmt.Errorf("error while loading relevant projects to deploy: %w", err)
	}

	sortedConfigs, err := sortConfigs(filteredProjects, loadedManifest.Environments.Names())
	if err != nil {
		return fmt.Errorf("error during configuration sort: %w", err)
	}

	logProjectsInfo(filteredProjects)
	logEnvironmentsInfo(loadedManifest.Environments)

	if err = doDeploy(sortedConfigs, loadedManifest.Environments, continueOnErr, dryRun); err != nil {
		return err
	}

	return nil
}

func doDeploy(configs project.ConfigsPerEnvironment, environments manifest.Environments, continueOnErr bool, dryRun bool) error {
	var deployErrs []error
	for envName, configs := range configs {
		logDeploymentInfo(dryRun, envName)
		env, found := environments[envName]

		if !found {
			if continueOnErr {
				deployErrs = append(deployErrs, fmt.Errorf("cannot find environment `%s`", envName))
				continue
			} else {
				return fmt.Errorf("cannot find environment `%s`", envName)
			}
		}

		dtClient, err := createDynatraceClient(env, dryRun)
		if err != nil {
			if continueOnErr {
				deployErrs = append(deployErrs, err)
				continue
			} else {
				return err
			}
		}

		errs := deploy.DeployConfigs(dtClient, api.NewAPIs(), configs, deploy.DeployConfigsOptions{
			ContinueOnErr: continueOnErr,
			DryRun:        dryRun,
		})
		deployErrs = append(deployErrs, errs...)
	}

	if deployErrs != nil {
		printErrorReport(deployErrs)
		return fmt.Errorf("errors during %s", getOperationNounForLogging(dryRun))
	}
	log.Info("%s finished without errors", getOperationNounForLogging(dryRun))
	return nil
}

func absPath(manifestPath string) (string, error) {
	manifestPath = filepath.Clean(manifestPath)
	return filepath.Abs(manifestPath)
}

func loadManifest(fs afero.Fs, manifestPath string, groups []string, environments []string) (*manifest.Manifest, error) {
	m, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
		Groups:       groups,
		Environments: environments,
	})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return nil, errors.New("error while loading manifest")
	}

	return &m, nil
}

func verifyClusterGen(environments manifest.Environments, dryRun bool) error {
	if !dryRun {
		err := cmdutils.VerifyClusterGen(environments)
		if err != nil {
			return err
		}
	}
	return nil
}

func loadProjects(fs afero.Fs, manifestPath string, man *manifest.Manifest) ([]project.Project, error) {
	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        *man,
		ParametersSerde: config.DefaultParameterParsers,
	})

	if errs != nil {
		printErrorReport(errs)
		return nil, errors.New("error while loading projects - you may be loading v1 projects, please 'convert' to v2")
	}

	return projects, nil
}

func filterProjects(projects []project.Project, specificProjects []string, specificEnvironments []string) ([]project.Project, error) {

	if len(specificProjects) > 0 {
		filtered, err := filterProjectsByName(projects, specificProjects)

		if err != nil {
			return nil, err
		}

		projectsWithDependencies, err := loadProjectsWithDependencies(projects, filtered, specificEnvironments)

		if err != nil {
			return nil, err
		}

		projects = projectsWithDependencies
	}

	return projects, nil
}

func sortConfigs(projects []project.Project, environmentNames []string) (project.ConfigsPerEnvironment, error) {
	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, environmentNames)
	if errs != nil {
		errutils.PrintErrors(errs)
		return nil, errors.New("error during sort")
	}
	return sortedConfigs, nil
}

func logProjectsInfo(projects []project.Project) {
	log.Info("Projects to be deployed:")
	for _, p := range projects {
		log.Info("  - %s", p)
	}
}

func logEnvironmentsInfo(environments manifest.Environments) {
	log.Info("Environments to deploy to:")
	for _, name := range environments.Names() {
		log.Info("  - %s", name)
	}
}
func logDeploymentInfo(dryRun bool, envName string) {
	if dryRun {
		log.Info("Validating configurations for environment `%s`...", envName)
	} else {
		log.Info("Deploying configurations to environment `%s`...", envName)
	}
}

func getOperationNounForLogging(dryRun bool) string {
	if dryRun {
		return "Validation"
	}
	return "Deployment"
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
		log.Error("=== General Errors ===")
		for _, err := range generalErrors {
			log.Error(errutils.ErrorString(err))
		}
	}

	groupedConfigErrors := groupConfigErrors(configErrors)

	for project, apiErrors := range groupedConfigErrors {
		for api, configErrors := range apiErrors {
			for config, errs := range configErrors {
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
					log.Error("%s:%s:%s %s", project, api, config, errutils.ErrorString(err))
				}

				for group, environmentErrors := range groupErrors {
					for env, errs := range environmentErrors {
						for _, err := range errs {
							log.Error("%s(%s) %s:%s:%s %T %s", env, group, project, api, config, err, errutils.ErrorString(err))
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

		typeErrors := projectErrors[coord.Project]

		if typeErrors == nil {
			typeErrors = make(ApiErrors)
			typeErrors[coord.Type] = make(ConfigErrors)
			projectErrors[coord.Project] = typeErrors
		}

		configErrors := typeErrors[coord.Type]

		if configErrors == nil {
			configErrors = make(ConfigErrors)
			typeErrors[coord.Type] = configErrors
		}

		configErrors[coord.ConfigId] = append(configErrors[coord.ConfigId], err)
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

func filterProjectsByName(projects []project.Project, names []string) ([]string, error) {
	var result []string

	foundProjects := map[string]struct{}{}

	for _, p := range projects {
		if containsName(names, p.Id) {
			foundProjects[p.Id] = struct{}{}
			result = append(result, p.Id)
		} else if containsName(names, p.GroupId) {
			foundProjects[p.GroupId] = struct{}{}
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
	return slices.Contains(names, name)
}

func createDynatraceClient(environment manifest.EnvironmentDefinition, dryRun bool) (client.Client, error) {
	if dryRun {
		return client.NewDummyClient(), nil
	}

	return client.NewDynatraceClient(client.NewTokenAuthClient(environment.Auth.Token.Value), environment.Url.Value, client.WithAutoServerVersion())
}
