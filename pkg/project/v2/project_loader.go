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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/loader"
	"github.com/spf13/afero"
	"slices"
	"strings"
)

type ProjectLoaderContext struct {
	KnownApis       map[string]struct{}
	WorkingDir      string
	Manifest        manifest.Manifest
	ParametersSerde map[string]parameter.ParameterSerDe
}

// DuplicateConfigIdentifierError occurs if configuration IDs are found more than once
type DuplicateConfigIdentifierError struct {
	// Location (coordinate) of the config.Config in whose ID overlaps with an existign one
	Location coordinate.Coordinate `json:"location"`
	// EnvironmentDetails of the environment for which the duplicate was loaded
	EnvironmentDetails configErrors.EnvironmentDetails `json:"environmentDetails"`
}

func (e DuplicateConfigIdentifierError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e DuplicateConfigIdentifierError) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e DuplicateConfigIdentifierError) Error() string {
	return fmt.Sprintf("Config IDs need to be unique to project/type, found duplicate `%s`", e.Location)
}

func newDuplicateConfigIdentifierError(c config.Config) DuplicateConfigIdentifierError {
	return DuplicateConfigIdentifierError{
		Location: c.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       c.Group,
			Environment: c.Environment,
		},
	}
}

func LoadProjects(fs afero.Fs, context ProjectLoaderContext, specificProjects []string) ([]Project, []error) {
	environments := toEnvironmentSlice(context.Manifest.Environments)
	var workingDirFs afero.Fs

	if context.WorkingDir == "." {
		workingDirFs = fs
	} else {
		workingDirFs = afero.NewBasePathFs(fs, context.WorkingDir)
	}

	if len(context.Manifest.Projects) == 0 {
		return nil, []error{fmt.Errorf("no projects defined in manifest")}
	}

	if len(specificProjects) == 0 {
		log.Info("Loading %d projects...", len(context.Manifest.Projects))
		return loadProjectsFromProjectDefinitions(workingDirFs, context, context.Manifest.Projects, environments)
	}

	specificProjectDefinitions, err := filterProjectDefinitionsByProjectNames(context.Manifest.Projects, specificProjects)
	if err != nil {
		return nil, []error{err}
	}
	log.Info("Loading %d projects...", len(specificProjectDefinitions))
	projects, errors := loadProjectsFromProjectDefinitions(workingDirFs, context, specificProjectDefinitions, environments)
	if errors != nil {
		return nil, errors
	}

	for {
		additionalDepedencyProjectNames := getAdditionalDependencyProjectNames(projects, environments)

		if len(additionalDepedencyProjectNames) == 0 {
			break
		}

		log.Info("Loading %d additional dependent projects...", len(additionalDepedencyProjectNames))
		dependencyProjectDefinitions, err := filterProjectDefinitionsByProjectNames(context.Manifest.Projects, additionalDepedencyProjectNames)
		if err != nil {
			return nil, []error{err}
		}

		dependencyProjects, errors := loadProjectsFromProjectDefinitions(workingDirFs, context, dependencyProjectDefinitions, environments)
		if errors != nil {
			return nil, errors
		}

		projects = append(projects, dependencyProjects...)
	}

	return projects, nil
}

func getAdditionalDependencyProjectNames(projects []Project, environments []manifest.EnvironmentDefinition) []string {
	seenProjectNames := map[string]struct{}{}
	for _, p := range projects {
		seenProjectNames[p.Id] = struct{}{}
	}

	additionalDependencyProjectNames := []string{}
	for _, p := range projects {
		for _, env := range environments {
			for _, d := range p.Dependencies[env.Name] {
				if _, found := seenProjectNames[d]; !found {
					seenProjectNames[d] = struct{}{}
					additionalDependencyProjectNames = append(additionalDependencyProjectNames, d)
				}
			}
		}
	}

	return additionalDependencyProjectNames
}

func filterProjectDefinitionsByProjectNames(projectDefinitions manifest.ProjectDefinitionByProjectID, projectNames []string) (manifest.ProjectDefinitionByProjectID, error) {
	filteredProjectDefinitions := make(manifest.ProjectDefinitionByProjectID, len(projectDefinitions))

	seenProjectNames := map[string]struct{}{}

	for projectName, projectDefinition := range projectDefinitions {
		if slices.Contains(projectNames, projectName) {
			filteredProjectDefinitions[projectName] = projectDefinition
			seenProjectNames[projectName] = struct{}{}
		} else if slices.Contains(projectNames, projectDefinition.Group) {
			filteredProjectDefinitions[projectName] = projectDefinition
			seenProjectNames[projectDefinition.Group] = struct{}{}
		}
	}

	var notSeenProjectNames []string
	for _, name := range projectNames {
		if _, found := seenProjectNames[name]; !found {
			notSeenProjectNames = append(notSeenProjectNames, name)
		}
	}

	if notSeenProjectNames != nil {
		return nil, fmt.Errorf("no project with names `%s` found", strings.Join(notSeenProjectNames, ", "))
	}

	return filteredProjectDefinitions, nil
}

func loadProjectsFromProjectDefinitions(workingDirFs afero.Fs, context ProjectLoaderContext, projectDefinitions manifest.ProjectDefinitionByProjectID, environments []manifest.EnvironmentDefinition) ([]Project, []error) {
	projects := make([]Project, 0)
	var errors []error
	for _, projectDefinition := range projectDefinitions {
		project, projectErrors := loadProject(workingDirFs, context, projectDefinition, environments)
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

	// find and memorize (non-unique-name) configurations with identical names and set a special parameter on them
	// to be able to identify them later
	// splitting is map[environment]map[name]count
	nonUniqueNameConfigCount := make(map[string]map[string]int)
	apis := api.NewAPIs()
	for _, c := range configs {
		if c.Type.ID() == config.ClassicApiTypeId && apis[c.Coordinate.Type].NonUniqueName {
			name, err := config.GetNameForConfig(c)
			if err != nil {
				log.WithFields(field.Error(err), field.Coordinate(c.Coordinate)).Error("Unable to resolve name of configuration")
			}

			if _, f := nonUniqueNameConfigCount[c.Environment]; !f {
				nonUniqueNameConfigCount[c.Environment] = make(map[string]int)
			}

			if nameStr, ok := name.(string); ok {
				nonUniqueNameConfigCount[c.Environment][nameStr]++
			}
		}
	}

	configMap := make(ConfigsPerTypePerEnvironments)
	for i, conf := range configs {
		name, _ := config.GetNameForConfig(configs[i])
		// set special parameter for non-unique configs that appear multiple times with the same name
		// in order to be able to identify them during deployment
		if nameStr, ok := name.(string); ok {
			if nonUniqueNameConfigCount[conf.Environment][nameStr] > 1 {
				configs[i].Parameters[config.NonUniqueNameConfigDuplicationParameter] = value.New(true)
			}
		}

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

func loadConfigsOfProject(fs afero.Fs, loadingContext ProjectLoaderContext, projectDefinition manifest.ProjectDefinition,
	environments []manifest.EnvironmentDefinition) ([]config.Config, []error) {

	configFiles, err := files.FindYamlFiles(fs, projectDefinition.Path)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to walk files: %w", err)}
	}

	var configs []config.Config
	var errs []error

	ctx := &loader.LoaderContext{
		ProjectId:       projectDefinition.Name,
		Environments:    environments,
		Path:            projectDefinition.Path,
		KnownApis:       loadingContext.KnownApis,
		ParametersSerDe: loadingContext.ParametersSerde,
	}

	for _, file := range configFiles {
		log.WithFields(field.F("file", file)).Debug("Loading configuration file %s", file)
		loadedConfigs, configErrs := loader.LoadConfigFile(fs, ctx, file)

		errs = append(errs, configErrs...)
		configs = append(configs, loadedConfigs...)
	}

	return configs, errs
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

func toDependenciesMap(projectId string, configs []config.Config) DependenciesPerEnvironment {
	result := make(DependenciesPerEnvironment)

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

			if !slices.Contains(result[c.Environment], ref.Project) {
				result[c.Environment] = append(result[c.Environment], ref.Project)
			}
		}
	}

	return result
}
