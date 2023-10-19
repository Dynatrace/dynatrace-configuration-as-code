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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// WriterContext holds all information for [WriteManifest]
type WriterContext struct {
	// Fs holds the abstraction of the file system.
	Fs afero.Fs

	// ManifestPath holds the path from where the manifest should be written to.
	ManifestPath string
}

type manifestWriterError struct {
	ManifestPath string `json:"manifestPath"`
	Err          error  `json:"error"`
}

func (e manifestWriterError) Unwrap() error {
	return e.Err
}

func (e manifestWriterError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Err)
}

func newManifestWriterError(path string, err error) manifestWriterError {
	return manifestWriterError{
		ManifestPath: path,
		Err:          err,
	}
}

func WriteManifest(context *WriterContext, manifestToWrite Manifest) error {
	sanitizedPath := filepath.Clean(context.ManifestPath)
	folder := filepath.Dir(sanitizedPath)

	if folder != "." {
		err := context.Fs.MkdirAll(folder, 0777)

		if err != nil {
			return newManifestWriterError(context.ManifestPath, err)
		}
	}

	projects := toWriteableProjects(manifestToWrite.Projects)
	groups := toWriteableEnvironmentGroups(manifestToWrite.Environments)

	manifestVersion := "1.0"
	if featureflags.AccountManagement().Enabled() {
		manifestVersion = version.ManifestVersion
	}

	m := persistence.Manifest{
		ManifestVersion:   manifestVersion,
		Projects:          projects,
		EnvironmentGroups: groups,
	}

	return persistManifestToDisk(context, m)
}

func persistManifestToDisk(context *WriterContext, m persistence.Manifest) error {
	manifestAsYaml, err := yaml.Marshal(m)

	if err != nil {
		return newManifestWriterError(context.ManifestPath, err)
	}

	err = afero.WriteFile(context.Fs, filepath.Clean(context.ManifestPath), manifestAsYaml, 0664)
	if err != nil {
		return newManifestWriterError(context.ManifestPath, err)
	}
	return nil
}

func toWriteableProjects(projects map[string]ProjectDefinition) (result []persistence.Project) {
	groups := map[string]persistence.Project{}

	for _, projectDefinition := range projects {

		if isGroupingProject(projectDefinition) {
			groupName, groupPath := extractGroupedProjectDetails(projectDefinition)

			groups[groupName] = persistence.Project{
				Name: groupName,
				Path: groupPath,
				Type: persistence.GroupProjectType,
			}
			continue
		}

		p := persistence.Project{Name: projectDefinition.Name}

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

func toWriteableEnvironmentGroups(environments map[string]EnvironmentDefinition) (result []persistence.Group) {
	environmentPerGroup := make(map[string][]persistence.Environment)

	for name, env := range environments {
		e := persistence.Environment{
			Name: name,
			URL:  toWriteableURL(env),
			Auth: getAuth(env),
		}

		environmentPerGroup[env.Group] = append(environmentPerGroup[env.Group], e)
	}

	for g, envs := range environmentPerGroup {
		result = append(result, persistence.Group{Name: g, Environments: envs})
	}

	return result
}

func getAuth(env EnvironmentDefinition) persistence.Auth {
	return persistence.Auth{
		Token: getTokenSecret(env.Auth, env.Name),
		OAuth: getOAuthCredentials(env.Auth.OAuth),
	}
}

func toWriteableURL(environment EnvironmentDefinition) persistence.Url {
	if environment.URL.Type == EnvironmentURLType {
		return persistence.Url{
			Type:  persistence.UrlTypeEnvironment,
			Value: environment.URL.Name,
		}
	}

	return persistence.Url{
		Value: environment.URL.Value,
	}
}

// getTokenSecret returns the tokenConfig with some legacy magic string append that still might be used (?)
func getTokenSecret(a Auth, envName string) persistence.AuthSecret {
	var envVarName string
	if a.Token.Name != "" {
		envVarName = a.Token.Name
	} else {
		envVarName = envName + "_TOKEN"
	}

	return persistence.AuthSecret{
		Type: persistence.TypeEnvironment,
		Name: envVarName,
	}
}

func getOAuthCredentials(a *OAuth) *persistence.OAuth {
	if a == nil {
		return nil
	}

	var te *persistence.Url
	if a.TokenEndpoint != nil {
		switch a.TokenEndpoint.Type {
		case ValueURLType:
			te = &persistence.Url{
				Value: a.TokenEndpoint.Value,
			}
		case EnvironmentURLType:
			te = &persistence.Url{
				Type:  persistence.UrlTypeEnvironment,
				Value: a.TokenEndpoint.Name,
			}
		}
	}

	return &persistence.OAuth{
		ClientID: persistence.AuthSecret{
			Type: persistence.TypeEnvironment,
			Name: a.ClientID.Name,
		},
		ClientSecret: persistence.AuthSecret{
			Type: persistence.TypeEnvironment,
			Name: a.ClientSecret.Name,
		},
		TokenEndpoint: te,
	}
}
