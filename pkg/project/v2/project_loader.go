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
	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"path/filepath"
	"strings"
)

type ProjectLoaderContext struct {
	Apis            []string
	WorkingDir      string
	Manifest        manifest.Manifest
	ParametersSerde map[string]parameter.ParameterSerDe
}

func LoadProjects(fs afero.Fs, context ProjectLoaderContext) ([]Project, []error) {
	environments := toEnvironmentSlice(context.Manifest.Environments)
	projects := make([]Project, 0)

	var workingDirFs afero.Fs

	if context.WorkingDir == "." {
		workingDirFs = fs
	} else {
		workingDirFs = afero.NewBasePathFs(fs, context.WorkingDir)
	}

	var errors []error

	for _, projectDefinition := range context.Manifest.Projects {
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

	log.Debug("Loading project `%s` (%s)...", projectDefinition.Name, projectDefinition.Path)

	configs := make([]config.Config, 0)
	var errors []error

	for _, api := range context.Apis {
		apiPath := filepath.Join(projectDefinition.Path, api)

		if exists, err := afero.Exists(fs, apiPath); !exists || err != nil {
			continue
		}

		if isDir, err := afero.IsDir(fs, apiPath); !isDir || err != nil {
			continue
		}

		loaded, configErrors := config.LoadConfigs(fs, &config.LoaderContext{
			ProjectId:       projectDefinition.Name,
			ApiId:           api,
			Path:            apiPath,
			Environments:    environments,
			ParametersSerDe: context.ParametersSerde,
		})

		if configErrors != nil {
			errors = append(errors, configErrors...)
			continue
		}

		configs = append(configs, loaded...)
	}

	if d := findDuplicatedConfigIdentifiers(configs); d != nil {
		errors = append(errors, fmt.Errorf("Config IDs need to be unique to project/type, found duplicates: [%s] ", strings.Join(d, ", ")))
	}

	if errors != nil {
		return Project{}, errors
	}

	configMap := make(ConfigsPerApisPerEnvironments)

	for _, conf := range configs {
		if _, found := configMap[conf.Environment]; !found {
			configMap[conf.Environment] = make(map[string][]config.Config)
		}

		configMap[conf.Environment][conf.Coordinate.Api] = append(configMap[conf.Environment][conf.Coordinate.Api], conf)
	}

	return Project{
		Id:           projectDefinition.Name,
		GroupId:      projectDefinition.Group,
		Configs:      configMap,
		Dependencies: toDependenciesMap(projectDefinition.Name, configs),
	}, nil
}

func findDuplicatedConfigIdentifiers(configs []config.Config) []string {

	coordinates := make(map[string]int)
	var duplicates []string
	for _, c := range configs {
		id := toFullyQualifiedConfigIdentifier(c)
		if timesFound, found := coordinates[id]; found && timesFound < 2 {
			duplicates = append(duplicates, c.Coordinate.String())
		}

		coordinates[id] += 1
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

		for _, ref := range c.References {
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
