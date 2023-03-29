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

package v2

import (
	"fmt"
	"os"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
)

type ProjectLoaderContext struct {
	KnownApis       map[string]struct{}
	WorkingDir      string
	Manifest        manifest.Manifest
	ParametersSerde map[string]parameter.ParameterSerDe
}

type DuplicateConfigIdentifierError struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
}

func (e DuplicateConfigIdentifierError) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e DuplicateConfigIdentifierError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e DuplicateConfigIdentifierError) Error() string {
	return fmt.Sprintf("Config IDs need to be unique to project/type, found duplicate `%s`", e.Config)
}

func newDuplicateConfigIdentifierError(c config.Config) DuplicateConfigIdentifierError {
	return DuplicateConfigIdentifierError{
		Config: c.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       c.Group,
			Environment: c.Environment,
		},
	}
}

func LoadProjects(fs afero.Fs, context ProjectLoaderContext) ([]Project, []error) {
	return LoadProjectsSpecific(fs, context, []string{}, []string{})
}

func LoadProjectsSpecificProjects(fs afero.Fs, context ProjectLoaderContext, specificProjects []string) ([]Project, []error) {
	return LoadProjectsSpecific(fs, context, specificProjects, []string{})
}

func LoadProjectsSpecificEnvironments(fs afero.Fs, context ProjectLoaderContext, specificEnvironments []string) ([]Project, []error) {
	return LoadProjectsSpecific(fs, context, []string{}, specificEnvironments)
}

func LoadProjectsSpecific(fs afero.Fs, context ProjectLoaderContext, specificProjects []string, specificEnvironments []string) ([]Project, []error) {
	filteredEnvironmentsSlice, err := filterRequiredEnvironments(context.Manifest.Environments, specificEnvironments)
	if err != nil {
		return nil, []error{err}
	}

	var workingDirFs afero.Fs

	if context.WorkingDir == "." {
		workingDirFs = fs
	} else {
		workingDirFs = afero.NewBasePathFs(fs, context.WorkingDir)
	}

	filteredProjects, err := filterRequiredProjects(context.Manifest.Projects, specificProjects)
	if err != nil {
		return nil, []error{err}
	}

	var errors []error
	projects := make([]Project, 0)
	for _, projectDefinition := range filteredProjects {

		project, projectErrors := loadProject(workingDirFs, context, projectDefinition, filteredEnvironmentsSlice)

		if projectErrors != nil {
			errors = append(errors, projectErrors...)
			continue
		}

		projects = append(projects, project)
	}

	if errors != nil {
		return nil, errors
	}

	return projects, nil
}

func filterRequiredEnvironments(environmentsMap map[string]manifest.EnvironmentDefinition, specificEnvironments []string) ([]manifest.EnvironmentDefinition, error) {

	environments := toEnvironmentSlice(environmentsMap)

	if len(specificEnvironments) == 0 {
		return environments, nil
	}

	filteredEnvironments := make([]manifest.EnvironmentDefinition, len(specificEnvironments))
	environementsFoundCount := 0

	for _, environmentDefinition := range environments {
		if slices.Contains(specificEnvironments, environmentDefinition.Name) {
			filteredEnvironments[environementsFoundCount] = environmentDefinition
			environementsFoundCount++
		}
	}

	if environementsFoundCount != len(specificEnvironments) {
		return nil, fmt.Errorf("only %d projects found in the manifest, requested projects: %v", environementsFoundCount, specificEnvironments)
	}

	return filteredEnvironments, nil
}

func filterRequiredProjects(projects manifest.ProjectDefinitionByProjectID, specificProjects []string) (manifest.ProjectDefinitionByProjectID, error) {

	if len(specificProjects) == 0 {
		return projects, nil
	}

	filteredProjects := manifest.ProjectDefinitionByProjectID{}
	projectFoundCount := 0

	for id, projectDefinition := range projects {
		if slices.Contains(specificProjects, projectDefinition.Name) {
			filteredProjects[id] = projectDefinition
			projectFoundCount++
		}
	}

	if projectFoundCount != len(specificProjects) {
		return nil, fmt.Errorf("only %d projects found in the manifest, requested projects: %v", projectFoundCount, specificProjects)
	}

	return filteredProjects, nil
}

func toEnvironmentSlice(environments map[string]manifest.EnvironmentDefinition) []manifest.EnvironmentDefinition {
	var result []manifest.EnvironmentDefinition

	for _, env := range environments {
		result = append(result, env)
	}

	return result
}

func loadProject(fs afero.Fs, context ProjectLoaderContext, projectDefinition manifest.ProjectDefinition,
	environments []manifest.EnvironmentDefinition) (Project, []error) {

	exists, err := afero.Exists(fs, projectDefinition.Path)
	if err != nil {
		return Project{}, []error{fmt.Errorf("failed to load project `%s` (%s): %w", projectDefinition.Name, projectDefinition.Path, err)}
	}
	if !exists {
		return Project{}, []error{fmt.Errorf("failed to load project `%s`: filepath `%s` does not exist", projectDefinition.Name, projectDefinition.Path)}
	}

	log.Debug("Loading project `%s` (%s)...", projectDefinition.Name, projectDefinition.Path)

	configs, errors := loadConfigsOfProject(fs, context, projectDefinition, environments)

	if d := findDuplicatedConfigIdentifiers(configs); d != nil {
		for _, c := range d {
			errors = append(errors, newDuplicateConfigIdentifierError(c))
		}
	}

	if errors != nil {
		return Project{}, errors
	}

	configMap := make(ConfigsPerTypePerEnvironments)

	for _, conf := range configs {
		if _, found := configMap[conf.Environment]; !found {
			configMap[conf.Environment] = make(map[string][]config.Config)
		}

		configMap[conf.Environment][conf.Coordinate.Type] = append(configMap[conf.Environment][conf.Coordinate.Type], conf)
	}

	return Project{
		Id:           projectDefinition.Name,
		GroupId:      projectDefinition.Group,
		Configs:      configMap,
		Dependencies: toDependenciesMap(projectDefinition.Name, configs),
	}, nil
}

func loadConfigsOfProject(fs afero.Fs, context ProjectLoaderContext, projectDefinition manifest.ProjectDefinition, environments []manifest.EnvironmentDefinition) ([]config.Config, []error) {
	var configs []config.Config
	var errors []error

	err := afero.Walk(fs, projectDefinition.Path, func(path string, info os.FileInfo, err error) error {

		if !info.IsDir() {
			return nil
		}

		pathParts := strings.Split(path, string(os.PathSeparator))
		anyHidden := slices.AnyMatches(pathParts, func(v string) bool {
			return strings.HasPrefix(v, ".")
		})

		if anyHidden {
			return nil
		}

		loaded, errs := config.LoadConfigs(fs, &config.LoaderContext{
			ProjectId:       projectDefinition.Name,
			Path:            path,
			Environments:    environments,
			KnownApis:       context.KnownApis,
			ParametersSerDe: context.ParametersSerde,
		})

		if errs != nil {
			errors = append(errors, errs...)
			return nil
		}

		configs = append(configs, loaded...)

		return nil
	})

	if err != nil {
		errors = append(errors, err)
	}

	return configs, errors
}

func findDuplicatedConfigIdentifiers(configs []config.Config) []config.Config {

	coordinates := make(map[string]struct{})
	var duplicates []config.Config
	for _, c := range configs {
		id := toFullyQualifiedConfigIdentifier(c)
		if _, found := coordinates[id]; found {
			duplicates = append(duplicates, c)
		}
		coordinates[id] = struct{}{}
	}
	return duplicates
}

// toFullyUniqueConfigIdentifier returns a configs coordinate as well as environment,
// as in the scope of project loader we might have "overlapping" coordinates for any loaded
// environment or group override of the same configuration
func toFullyQualifiedConfigIdentifier(config config.Config) string {
	return fmt.Sprintf("%s:%s:%s", config.Group, config.Environment, config.Coordinate)
}

func toDependenciesMap(projectId string,
	configs []config.Config) map[string][]string {
	result := make(map[string][]string)

	for _, c := range configs {
		// ignore skipped configs
		if c.Skip {
			continue
		}

		for _, ref := range c.References() {
			// ignore project on same project
			if projectId == ref.Project {
				continue
			}

			if !containsProject(result[c.Environment], ref.Project) {
				result[c.Environment] = append(result[c.Environment], ref.Project)
			}
		}
	}

	return result
}

func containsProject(projects []string, project string) bool {
	for _, p := range projects {
		if p == project {
			return true
		}
	}

	return false
}
