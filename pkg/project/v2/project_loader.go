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
	ref "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/config/loader"
	"github.com/spf13/afero"
	"slices"
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

// Tries to load the specified projects. If no project names are specified, all projects are loaded.
func LoadProjects(fs afero.Fs, context ProjectLoaderContext, specificProjectNames []string) ([]Project, []error) {
	var workingDirFs afero.Fs

	if context.WorkingDir == "." {
		workingDirFs = fs
	} else {
		workingDirFs = afero.NewBasePathFs(fs, context.WorkingDir)
	}

	if len(context.Manifest.Projects) == 0 {
		return nil, []error{fmt.Errorf("no projects defined in manifest")}
	}

	environments := toEnvironmentSlice(context.Manifest.Environments)

	projectNamesToLoad, errors := getProjectNamesToLoad(context.Manifest.Projects, specificProjectNames)

	seenProjectNames := make(map[string]struct{}, len(projectNamesToLoad))
	var loadedProjects []Project
	for len(projectNamesToLoad) > 0 {
		projectNameToLoad := projectNamesToLoad[0]
		projectNamesToLoad = projectNamesToLoad[1:]

		if _, found := seenProjectNames[projectNameToLoad]; found {
			continue
		}
		seenProjectNames[projectNameToLoad] = struct{}{}

		projectDefinition, found := context.Manifest.Projects[projectNameToLoad]
		if !found {
			continue
		}

		project, errs := loadProject(workingDirFs, context, projectDefinition, environments)
		if len(errs) > 0 {
			errors = append(errors, errs...)
			continue
		}

		loadedProjects = append(loadedProjects, project)

		for _, environment := range environments {
			projectNamesToLoad = append(projectNamesToLoad, project.Dependencies[environment.Name]...)
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	return loadedProjects, nil
}

// Gets full project names to load specified by project or grouping project names. If none are specified, all project names are returned. Errors are returned for any project names that do not exist.
func getProjectNamesToLoad(allProjectsDefinitions manifest.ProjectDefinitionByProjectID, specificProjectNames []string) ([]string, []error) {
	projectNamesToLoad := make([]string, 0, len(specificProjectNames))

	// if no projects are specified, all projects should be loaded
	if len(specificProjectNames) == 0 {
		for projectId := range allProjectsDefinitions {
			projectNamesToLoad = append(projectNamesToLoad, projectId)
		}
		return projectNamesToLoad, nil
	}

	var errors []error
	for _, projectName := range specificProjectNames {
		// try to find a project with the given name
		if _, found := allProjectsDefinitions[projectName]; found {
			projectNamesToLoad = append(projectNamesToLoad, projectName)
			continue
		}

		// try to find projects in a grouping project with the given name
		found := false
		for _, projectDefinition := range allProjectsDefinitions {
			if projectDefinition.Group == projectName {
				projectNamesToLoad = append(projectNamesToLoad, projectDefinition.Name)
				found = true
			}
		}

		if !found {
			errors = append(errors, fmt.Errorf("no project named `%s` could be found in the manifest", projectName))
		}
	}

	return projectNamesToLoad, errors
}

func toEnvironmentSlice(environments map[string]manifest.EnvironmentDefinition) []manifest.EnvironmentDefinition {
	var result []manifest.EnvironmentDefinition

	for _, env := range environments {
		result = append(result, env)
	}

	return result
}

func loadProject(fs afero.Fs, context ProjectLoaderContext, projectDefinition manifest.ProjectDefinition, environments []manifest.EnvironmentDefinition) (Project, []error) {
	exists, err := afero.Exists(fs, projectDefinition.Path)
	if err != nil {
		return Project{}, []error{fmt.Errorf("failed to load project `%s` (%s): %w", projectDefinition.Name, projectDefinition.Path, err)}
	}
	if !exists {
		return Project{}, []error{fmt.Errorf("failed to load project `%s`: filepath `%s` does not exist", projectDefinition.Name, projectDefinition.Path)}
	}

	log.Debug("Loading project `%s` (%s)...", projectDefinition.Name, projectDefinition.Path)

	configs, errors := loadConfigsOfProject(fs, context, projectDefinition, environments)
	errors = append(errors, findDuplicatedConfigIdentifiers(configs)...)
	errors = append(errors, checkKeyUserActionScope(configs)...)

	if len(errors) > 0 {
		return Project{}, errors
	}

	insertNetworkZoneParameter(configs)

	return Project{
		Id:           projectDefinition.Name,
		GroupId:      projectDefinition.Group,
		Configs:      toConfigMap(configs),
		Dependencies: toDependenciesMap(projectDefinition.Name, configs),
	}, nil
}

// insertNetworkZoneParameter ensures that the “builtin:networkzone” settings 2.0 objects are deployed prior to any
// “networkzone” configurations. This is crucial because “builtin:networkzone” is responsible for activating the network
// zone features. If these are not deployed before any actual “networkzone” configuration, it could potentially lead to an error.
// This function ensures that if “networkzones” and “builtin:networkzones” settings 2.0 objects are found, a dependency is
// created in the form of a reference parameter that points to the “builtin:networkzone” configuration for each networkzone
// configuration. This dependency ensures the correct order of deployment.
func insertNetworkZoneParameter(configs []config.Config) {
	var networkZones []config.Config
	var networkZoneEnabled config.Config
	var networkZoneEnabledFound bool
	for _, c := range configs {
		if c.Coordinate.Type == api.NetworkZone {
			networkZones = append(networkZones, c)
		}
		if c.Coordinate.Type == "builtin:networkzones" {
			networkZoneEnabled = c
			networkZoneEnabledFound = true
		}
	}
	// Note: Adding a parameter to an already existing parameter (e.g. created by the user) is redundant but does no harm
	if len(networkZones) > 0 && networkZoneEnabledFound {
		for _, nz := range networkZones {
			nz.Parameters["__MONACO_NZONE_ENABLED__"] = &ref.ReferenceParameter{
				ParameterReference: parameter.ParameterReference{Config: networkZoneEnabled.Coordinate, Property: "name"}}
		}
	}
}

func checkKeyUserActionScope(configs []config.Config) []error {
	var errors []error
	for _, c := range configs {
		// The scope parameter of a key user actions web configuration needs to be a reference to another application-web config
		// The reference parameter makes sure that rely on the fact that kua web configs are loaded/deployed within the same
		// sub graph (independent component) later on as
		if c.Coordinate.Type == api.KeyUserActionsWeb {
			if _, ok := c.Parameters[config.ScopeParameter].(*ref.ReferenceParameter); !ok {
				errors = append(errors, fmt.Errorf("scope parameter of config of type '%s' with ID '%s' needs to be a reference "+
					"parameter to another web-application config", api.KeyUserActionsWeb, c.Coordinate.ConfigId))
			}
		}
	}
	return errors
}

func toConfigMap(configs []config.Config) ConfigsPerTypePerEnvironments {
	// find and memorize (non-unique-name) configurations with identical names and set a special parameter on them
	// to be able to identify them later
	// splitting is map[environment]map[name]count
	nonUniqueNameConfigCount := make(map[string]map[string]int)
	apis := api.NewAPIs()
	for _, c := range configs {
		if c.Type.ID() == config.ClassicApiTypeID && apis[c.Coordinate.Type].NonUniqueName {
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
	return configMap
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

	// TODO ugly and produces memory overhead, but the easiest way to make global variables available is putting them right in the config
	// where they can then be referenced via ref params to "property: "_global:<key>""
	// Rather than magic strings, a proper implementation should IMO introduce a 'global' parameter type for configs and make access explicit
	for i, _ := range configs {
		for k, v := range loadingContext.Manifest.Parameters {
			configs[i].Parameters["_global:"+k] = v
		}
		configs[i].Parameters["_global:environment"] = value.New(configs[i].Environment)
	}

	return configs, errs
}

func findDuplicatedConfigIdentifiers(configs []config.Config) []error {
	var errors []error
	coordinates := make(map[string]struct{})
	for _, c := range configs {
		id := toFullyQualifiedConfigIdentifier(c)
		if _, found := coordinates[id]; found {
			errors = append(errors, newDuplicateConfigIdentifierError(c))
		}
		coordinates[id] = struct{}{}
	}
	return errors
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
