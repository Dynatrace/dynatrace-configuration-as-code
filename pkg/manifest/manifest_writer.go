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

package manifest

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type ManifestWriterContext struct {
	Fs           afero.Fs
	ManifestPath string
}

func WriteManifest(context *ManifestWriterContext, manifestToWrite Manifest) error {
	sanitizedPath := filepath.Clean(context.ManifestPath)
	folder := filepath.Dir(sanitizedPath)

	if folder != "." {
		err := context.Fs.MkdirAll(folder, 0777)

		if err != nil {
			return err
		}
	}

	projects := toWriteableProjects(manifestToWrite.Projects)
	groups := toWriteableEnvironmentGroups(manifestToWrite.Environments)

	m := manifest{
		ManifestVersion:   version.ManifestVersion,
		Projects:          projects,
		EnvironmentGroups: groups,
	}

	return persistManifestToDisk(context, m)
}

func persistManifestToDisk(context *ManifestWriterContext, m manifest) error {
	manifestAsYaml, err := yaml.Marshal(m)

	if err != nil {
		return err
	}

	return afero.WriteFile(context.Fs, filepath.Clean(context.ManifestPath), manifestAsYaml, 0664)
}

func toWriteableProjects(projects map[string]ProjectDefinition) (result []project) {
	groups := map[string]project{}

	for _, projectDefinition := range projects {

		if isGroupingProject(projectDefinition) {
			groupName, groupPath := extractGroupedProjectDetails(projectDefinition)

			groups[groupName] = project{
				Name: groupName,
				Path: groupPath,
				Type: groupProjectType,
			}
			continue
		}

		p := project{Name: projectDefinition.Name}

		if projectDefinition.Name != projectDefinition.Path {
			p.Path = projectDefinition.Path
		}

		result = append(result, p)
	}

	for _, projectGroup := range groups {
		result = append(result, projectGroup)
	}

	return result
}

func isGroupingProject(projectDefinition ProjectDefinition) bool {
	return strings.Contains(projectDefinition.Name, ".") &&
		strings.ReplaceAll(projectDefinition.Name, ".", "/") == projectDefinition.Path
}

func extractGroupedProjectDetails(projectDefinition ProjectDefinition) (groupName, groupPath string) {
	subgroups := strings.Split(projectDefinition.Name, ".")
	projectName := subgroups[len(subgroups)-1]
	groupName = strings.TrimSuffix(projectDefinition.Name, "."+projectName)
	groupPath = strings.TrimSuffix(projectDefinition.Path, "/"+projectName)

	return groupName, groupPath
}

func toWriteableEnvironmentGroups(environments map[string]EnvironmentDefinition) (result []group) {
	environmentPerGroup := make(map[string][]environment)

	for name, env := range environments {
		e := environment{
			Name:  name,
			Url:   toWriteableUrl(env),
			Token: toWritableToken(env),
		}

		environmentPerGroup[env.Group] = append(environmentPerGroup[env.Group], e)
	}

	for g, envs := range environmentPerGroup {
		result = append(result, group{Name: g, Environments: envs})
	}

	return result
}

func toWriteableUrl(environment EnvironmentDefinition) url {
	if environment.url.Type == EnvironmentUrlType {
		return url{
			Type:  string(environment.url.Type),
			Value: environment.url.Value,
		}
	}

	return url{
		Value: environment.url.Value,
	}
}

func toWritableToken(environment EnvironmentDefinition) tokenConfig {
	token := environment.Name + "_TOKEN"

	if environment.Token.Name != "" {
		token = environment.Token.Name
	}

	return tokenConfig{
		Config: map[string]interface{}{
			"name": token,
		},
	}
}
