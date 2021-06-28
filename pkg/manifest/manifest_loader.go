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
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type ManifestLoaderContext struct {
	Fs           afero.Fs
	ManifestPath string
}

type projectLoaderContext struct {
	fs           afero.Fs
	manifestPath string
}

type ManifestLoaderError struct {
	ManifestPath string
	Reason       string
}

func (e *ManifestLoaderError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Reason)
}

type ManifestEnvironmentLoaderError struct {
	ManifestLoaderError
	Group       string
	Environment string
}

func (e *ManifestEnvironmentLoaderError) Error() string {
	return fmt.Sprintf("%s:%s:%s: %s", e.ManifestPath, e.Group, e.Environment, e.Reason)
}

type ManifestProjectLoaderError struct {
	ManifestLoaderError
	Project string
}

func (e *ManifestProjectLoaderError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.ManifestPath, e.Project, e.Reason)
}

func LoadManifest(context *ManifestLoaderContext) (Manifest, []error) {
	manifestPath := filepath.Clean(context.ManifestPath)

	if !files.IsYaml(manifestPath) {
		return Manifest{}, []error{
			&ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       "manifest file is not a yaml",
			},
		}
	}

	exists, err := files.DoesFileExist(context.Fs, manifestPath)

	if err != nil {
		return Manifest{}, []error{err}
	}

	if !exists {
		return Manifest{}, []error{
			&ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       "specified manifest file is either no file or does not exist",
			},
		}
	}

	data, err := afero.ReadFile(context.Fs, manifestPath)

	if err != nil {
		return Manifest{}, []error{
			&ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       fmt.Sprintf("error while reading the manifest: %s", err),
			},
		}
	}

	return parseManifest(context, data)
}

func parseManifest(context *ManifestLoaderContext, data []byte) (Manifest, []error) {
	manifestPath := filepath.Clean(context.ManifestPath)
	var manifest manifest

	err := yaml.Unmarshal(data, &manifest)

	if err != nil {
		return Manifest{}, []error{
			&ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       fmt.Sprintf("error during parsing of the manifest: `%s`", err),
			},
		}
	}

	var errors []error

	workingDir := filepath.Dir(manifestPath)
	var workingDirFs afero.Fs

	if workingDir == "." {
		workingDirFs = context.Fs
	} else {
		workingDirFs = afero.NewBasePathFs(context.Fs, workingDir)
	}

	relativeManifestPath := filepath.Base(manifestPath)

	projectDefinitions, projectErrors := toProjectDefinitions(&projectLoaderContext{
		fs:           workingDirFs,
		manifestPath: relativeManifestPath,
	}, manifest.Projects)

	if projectErrors != nil {
		errors = append(errors, projectErrors...)
	} else if len(projectDefinitions) == 0 {
		errors = append(errors, &ManifestLoaderError{
			ManifestPath: context.ManifestPath,
			Reason:       "no projects defined in manifest",
		})
	}

	environmentDefinitions, manifestErrors := toEnvironments(context, manifest.Environments)

	if manifestErrors != nil {
		errors = append(errors, manifestErrors...)
	} else if len(environmentDefinitions) == 0 {
		errors = append(errors, &ManifestLoaderError{
			ManifestPath: context.ManifestPath,
			Reason:       "no environments defined in manifest",
		})
	}

	if errors != nil {
		return Manifest{}, errors
	}

	return Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
	}, nil
}

func toEnvironments(context *ManifestLoaderContext, groups []group) (map[string]EnvironmentDefinition, []error) {
	var errors []error
	environments := make(map[string]EnvironmentDefinition)

	for i, group := range groups {
		if group.Group == "" {
			errors = append(errors, &ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       fmt.Sprintf("missing group name on index `%d`", i),
			})
			continue
		}

		for _, conf := range group.Entries {
			env, configErrors := toEnvironment(context, conf, group.Group)

			if configErrors != nil {
				errors = append(errors, configErrors...)
				continue
			}

			if _, found := environments[env.Name]; found {
				errors = append(errors, &ManifestLoaderError{
					ManifestPath: context.ManifestPath,
					Reason:       fmt.Sprintf("environment with name `%s` already exists", env.Name),
				})
				continue
			}

			environments[env.Name] = env
		}
	}

	if errors != nil {
		return nil, errors
	}

	return environments, nil
}

func toEnvironment(context *ManifestLoaderContext, config environment, group string) (EnvironmentDefinition, []error) {
	var errors []error

	token, err := parseToken(context, config, group, config.Token)

	if err != nil {
		errors = append(errors, err)
	}

	if config.Url == "" {
		errors = append(errors, &ManifestEnvironmentLoaderError{
			ManifestLoaderError: ManifestLoaderError{
				ManifestPath: context.ManifestPath,
				Reason:       "no `url` configured or value is blank",
			},
			Group:       group,
			Environment: config.Name,
		})
	}

	if len(errors) > 0 {
		return EnvironmentDefinition{}, errors
	}

	return EnvironmentDefinition{
		Name:  config.Name,
		Url:   config.Url,
		Token: token,
		Group: group,
	}, nil
}

func parseToken(context *ManifestLoaderContext, config environment, group string, token tokenConfig) (Token, error) {
	var tokenType string

	if token.Type == nil {
		tokenType = "environment"
	} else {
		tokenType = *token.Type
	}

	switch tokenType {
	case "environment":
		return parseEnvironmentToken(context, group, config, token)
	}

	return nil, &ManifestEnvironmentLoaderError{
		ManifestLoaderError: ManifestLoaderError{
			ManifestPath: context.ManifestPath,
			Reason:       fmt.Sprintf("unknwon token type `%s`", tokenType),
		},
		Group:       group,
		Environment: config.Name,
	}
}

func parseEnvironmentToken(context *ManifestLoaderContext, group string, config environment, token tokenConfig) (Token, error) {
	if val, found := token.Config["name"]; found {
		return &EnvironmentVariableToken{
			EnvironmentVariableName: util.ToString(val),
		}, nil
	}

	return nil, &ManifestEnvironmentLoaderError{
		ManifestLoaderError: ManifestLoaderError{
			ManifestPath: context.ManifestPath,
			Reason:       "missing key `name` in token config",
		},
		Group:       group,
		Environment: config.Name,
	}
}

func toProjectDefinitions(context *projectLoaderContext, definitions []project) (map[string]ProjectDefinition, []error) {
	var errors []error
	result := make(map[string]ProjectDefinition)

	for _, project := range definitions {
		parsed, projectErrors := parseProjectDefinition(context, project)

		if projectErrors != nil {
			errors = append(errors, projectErrors...)
			continue
		}

		for _, project := range parsed {
			if _, found := result[project.Name]; found {
				errors = append(errors, &ManifestLoaderError{
					ManifestPath: context.manifestPath,
					Reason:       fmt.Sprintf("duplicated project name `%s`", project.Name),
				})

				continue
			}

			result[project.Name] = project
		}
	}

	if errors != nil {
		return nil, errors
	}

	return result, nil
}

func parseProjectDefinition(context *projectLoaderContext, project project) ([]ProjectDefinition, []error) {
	var projectType string

	if project.Type == nil {
		projectType = "simple"
	} else {
		projectType = *project.Type
	}

	switch projectType {
	case "simple":
		return parseSimpleProjectDefinition(context, project)
	case "grouping":
		return parseGroupingProjectDefinition(context, project)
	default:
		return nil, []error{
			&ManifestProjectLoaderError{
				ManifestLoaderError: ManifestLoaderError{
					ManifestPath: context.manifestPath,
					Reason:       fmt.Sprintf("invalid project type `%s`", projectType),
				},
				Project: project.Name,
			},
		}
	}
}

func parseSimpleProjectDefinition(context *projectLoaderContext, project project) ([]ProjectDefinition, []error) {
	if project.Path == "" && project.Name == "" {
		return nil, []error{
			&ManifestProjectLoaderError{
				ManifestLoaderError: ManifestLoaderError{
					ManifestPath: context.manifestPath,
					Reason:       "project is missing both name and path",
				},
			},
		}
	}

	if strings.ContainsAny(project.Name, `/\`) {
		return nil, []error{
			&ManifestProjectLoaderError{
				ManifestLoaderError: ManifestLoaderError{
					ManifestPath: context.manifestPath,
					Reason:       `project name is not allowed to contain '/' or '\'`,
				},
				Project: project.Name,
			},
		}
	}

	if project.Path == "" {
		return []ProjectDefinition{
			{
				Name: project.Name,
				Path: project.Name,
			},
		}, nil
	}

	return []ProjectDefinition{
		{
			Name: project.Name,
			Path: project.Path,
		},
	}, nil
}

func parseGroupingProjectDefinition(context *projectLoaderContext, project project) ([]ProjectDefinition, []error) {
	projectPath := filepath.FromSlash(project.Path)

	files, err := afero.ReadDir(context.fs, projectPath)

	if err != nil {
		return nil, []error{err}
	}

	var result []ProjectDefinition

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		result = append(result, ProjectDefinition{
			Name: project.Name + "." + file.Name(),
			Path: filepath.Join(projectPath, file.Name()),
		})
	}

	if result == nil {
		// TODO should we really fail here?
		return nil, []error{
			&ManifestProjectLoaderError{
				ManifestLoaderError: ManifestLoaderError{
					ManifestPath: context.manifestPath,
					Reason:       fmt.Sprintf("no projects found in `%s`", projectPath),
				},
				Project: project.Name,
			},
		}
	}

	return result, nil
}
