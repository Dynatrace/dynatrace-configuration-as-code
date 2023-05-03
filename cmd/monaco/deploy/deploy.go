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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/deploy"
	"path/filepath"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
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

	ok := verifyEnvironmentGen(loadedManifest.Environments, dryRun)
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
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

	if err := checkEnvironments(sortedConfigs, loadedManifest.Environments); err != nil {
		return err
	}

	logProjectsInfo(filteredProjects)
	logEnvironmentsInfo(loadedManifest.Environments)

	var deployErrs []error
	for envName, cfgs := range sortedConfigs {
		env := loadedManifest.Environments[envName]
		errs := deployOnEnvironment(&env, cfgs, continueOnErr, dryRun)
		deployErrs = append(deployErrs, errs...)
		if len(errs) > 0 && !continueOnErr {
			break
		}
	}

	if len(deployErrs) > 0 {
		printErrorReport(deployErrs)
		return fmt.Errorf("errors during %s", getOperationNounForLogging(dryRun))
	}
	log.Info("%s finished without errors", getOperationNounForLogging(dryRun))
	return nil
}

func deployOnEnvironment(env *manifest.EnvironmentDefinition, cfgs []config.Config, continueOnErr bool, dryRun bool) []error {
	logDeploymentInfo(dryRun, env.Name)
	clientSet, _ := deploy.CreateClientSet(env.URL.Value, env.Auth, dryRun)
	errs := deploy.DeployConfigs(clientSet, api.NewAPIs(), cfgs, deploy.DeployConfigsOptions{
		ContinueOnErr: continueOnErr,
		DryRun:        dryRun,
	})
	return errs
}

func absPath(manifestPath string) (string, error) {
	manifestPath = filepath.Clean(manifestPath)
	return filepath.Abs(manifestPath)
}

func loadManifest(fs afero.Fs, manifestPath string, groups []string, environments []string) (*manifest.Manifest, error) {
	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
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

func verifyEnvironmentGen(environments manifest.Environments, dryRun bool) bool {
	if !dryRun {
		return dynatrace.VerifyEnvironmentGeneration(environments)

	}
	return true
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

func filterProjectsByName(projects []project.Project, names []string) ([]string, error) {
	var result []string

	foundProjects := map[string]struct{}{}

	for _, p := range projects {
		if slices.Contains(names, p.Id) {
			foundProjects[p.Id] = struct{}{}
			result = append(result, p.Id)
		} else if slices.Contains(names, p.GroupId) {
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

func checkEnvironments(configs project.ConfigsPerEnvironment, envs manifest.Environments) error {
	for envName, cfgs := range configs {
		if _, found := envs[envName]; !found {
			return fmt.Errorf("cannot find environment `%s`", envName)
		}
		if err := checkConfigsForEnvironment(envs[envName], cfgs); err != nil {
			return err
		}
	}
	return nil
}

func checkConfigsForEnvironment(env manifest.EnvironmentDefinition, cfgs []config.Config) error {
	for i := range cfgs {
		if !cfgs[i].Skip && onlyAvailableOnPlatform(&cfgs[i]) && !platformEnvironment(env) {
			return fmt.Errorf("enviroment %q is not specified as platform, but at least one of configurations (e.g. %q) is platform exclusive", env.Name, cfgs[i].Coordinate)
		}
	}
	return nil
}

func platformEnvironment(e manifest.EnvironmentDefinition) bool {
	return e.Auth.OAuth != nil
}

func onlyAvailableOnPlatform(c *config.Config) bool {
	_, aut := c.Type.(config.AutomationType)
	return aut
}
