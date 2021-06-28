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
	"path/filepath"

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
		Projects:     projects,
		Environments: groups,
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

func toWriteableProjects(projects map[string]ProjectDefinition) []project {
	var result []project

	for projectName, projectDefinition := range projects {
		p := project{
			Name: projectName,
			Path: projectDefinition.Path,
		}

		result = append(result, p)
	}

	return result
}

func toWriteableEnvironmentGroups(environments map[string]EnvironmentDefinition) []group {
	environmentPerGroup := make(map[string][]environment)

	for name, env := range environments {
		e := environment{
			Name:  name,
			Url:   env.Url,
			Token: toWritableToken(env),
		}

		environmentPerGroup[env.Group] = append(environmentPerGroup[env.Group], e)
	}

	var result []group

	for g, envs := range environmentPerGroup {
		result = append(result, group{
			Group:   g,
			Entries: envs,
		})
	}

	return result
}

func toWritableToken(environment EnvironmentDefinition) tokenConfig {
	var envVarName string

	switch token := environment.Token.(type) {
	case *EnvironmentVariableToken:
		envVarName = token.EnvironmentVariableName
	default:
		envVarName = environment.Name + "_TOKEN"
	}

	return tokenConfig{
		Config: map[string]interface{}{
			"name": envVarName,
		},
	}
}
