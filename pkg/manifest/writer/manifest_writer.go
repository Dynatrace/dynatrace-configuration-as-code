/*
 * @license
 * Copyright 2023 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package writer

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Context holds all information for [Write]
type Context struct {
	// Fs holds the abstraction of the file system.
	Fs afero.Fs

	// ManifestPath holds the path from where the manifest should be written to.
	ManifestPath string
}

type ManifestWriterError struct {
	ManifestPath string `json:"manifestPath"`
	Err          error  `json:"error"`
}

func (e ManifestWriterError) Unwrap() error {
	return e.Err
}

func (e ManifestWriterError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Err)
}

func newManifestWriterError(path string, err error) ManifestWriterError {
	return ManifestWriterError{
		ManifestPath: path,
		Err:          err,
	}
}

// Write writes the manifest to the given path
func Write(context *Context, manifestToWrite manifest.Manifest) error {
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

	m := persistence.Manifest{
		ManifestVersion:   version.ManifestVersion,
		Projects:          projects,
		EnvironmentGroups: groups,
	}

	m.Accounts = toWriteableAccounts(manifestToWrite.Accounts)

	return persistManifestToDisk(context, m)
}

func persistManifestToDisk(context *Context, m persistence.Manifest) error {
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

func toWriteableProjects(projects map[string]manifest.ProjectDefinition) (result []persistence.Project) {
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

func isGroupingProject(projectDefinition manifest.ProjectDefinition) bool {
	return strings.Contains(projectDefinition.Name, ".") &&
		strings.ReplaceAll(projectDefinition.Name, ".", "/") == projectDefinition.Path
}

func extractGroupedProjectDetails(projectDefinition manifest.ProjectDefinition) (groupName, groupPath string) {
	subgroups := strings.Split(projectDefinition.Name, ".")
	projectName := subgroups[len(subgroups)-1]
	groupName = strings.TrimSuffix(projectDefinition.Name, "."+projectName)
	groupPath = strings.TrimSuffix(projectDefinition.Path, "/"+projectName)

	return groupName, groupPath
}

func toWriteableEnvironmentGroups(environments map[string]manifest.EnvironmentDefinition) (result []persistence.Group) {
	environmentPerGroup := make(map[string][]persistence.Environment)

	for name, env := range environments {
		e := persistence.Environment{
			Name: name,
			URL:  toWriteableURL(env.URL),
			Auth: getAuth(env),
		}

		environmentPerGroup[env.Group] = append(environmentPerGroup[env.Group], e)
	}

	for g, envs := range environmentPerGroup {
		result = append(result, persistence.Group{Name: g, Environments: envs})
	}

	return result
}

func getAuth(env manifest.EnvironmentDefinition) persistence.Auth {
	return persistence.Auth{
		Token: getTokenSecret(env.Auth, env.Name),
		OAuth: getOAuthCredentials(env.Auth.OAuth),
	}
}

func toWriteableURL(url manifest.URLDefinition) persistence.TypedValue {
	if url.Type == manifest.EnvironmentURLType {
		return persistence.TypedValue{
			Type:  persistence.TypeEnvironment,
			Value: url.Name,
		}
	}

	return persistence.TypedValue{
		Value: url.Value,
	}
}

// getTokenSecret returns the tokenConfig with some legacy magic string append that still might be used (?)
func getTokenSecret(a manifest.Auth, envName string) *persistence.AuthSecret {
	if a.Token == nil {
		return nil
	}

	envVarName := a.Token.Name
	if envVarName == "" {
		envVarName = envName + "_TOKEN"
	}

	return &persistence.AuthSecret{
		Type: persistence.TypeEnvironment,
		Name: envVarName,
	}
}

func getOAuthCredentials(a *manifest.OAuth) *persistence.OAuth {
	if a == nil {
		return nil
	}

	var te *persistence.TypedValue
	if a.TokenEndpoint != nil {
		url := toWriteableURL(*a.TokenEndpoint)
		te = &url
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

func toWriteableAccounts(accounts map[string]manifest.Account) []persistence.Account {
	var out []persistence.Account
	for _, account := range accounts {

		var apiURL *persistence.TypedValue
		if account.ApiUrl != nil {
			url := toWriteableURL(*account.ApiUrl)
			apiURL = &url
		}

		oauth := persistence.OAuth{
			ClientID: persistence.AuthSecret{
				Type: persistence.TypeEnvironment,
				Name: account.OAuth.ClientID.Name,
			},
			ClientSecret: persistence.AuthSecret{
				Type: persistence.TypeEnvironment,
				Name: account.OAuth.ClientSecret.Name,
			},
		}
		if account.OAuth.TokenEndpoint != nil {
			url := toWriteableURL(*account.OAuth.TokenEndpoint)
			oauth.TokenEndpoint = &url
		}

		out = append(out, persistence.Account{
			Name:        account.Name,
			AccountUUID: persistence.TypedValue{Value: account.AccountUUID.String()},
			ApiUrl:      apiURL,
			OAuth:       oauth,
		})
	}
	return out
}
