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
	"path/filepath"

	config "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/spf13/afero"
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
		Configs:      configMap,
		Dependencies: toDependenciesMap(projectDefinition.Name, configs),
	}, nil
}

func toDependenciesMap(projectId string,
	configs []config.Config) map[string][]string {
	result := make(map[string][]string)

	for _, config := range configs {
		// ignore skipped configs
		if config.Skip {
			continue
		}

		for _, ref := range config.References {
			// ignore project on same project
			if projectId == ref.Project {
				continue
			}

			if !containsProject(result[config.Environment], ref.Project) {
				result[config.Environment] = append(result[config.Environment], ref.Project)
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
