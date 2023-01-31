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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
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

func newManifestLoaderError(manifest string, reason string) ManifestLoaderError {
	return ManifestLoaderError{
		ManifestPath: manifest,
		Reason:       reason,
	}
}

func (e ManifestLoaderError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Reason)
}

type ManifestEnvironmentLoaderError struct {
	ManifestLoaderError
	Group       string
	Environment string
}

func newManifestEnvironmentLoaderError(manifest string, group string, env string, reason string) ManifestEnvironmentLoaderError {
	return ManifestEnvironmentLoaderError{
		ManifestLoaderError: newManifestLoaderError(manifest, reason),
		Group:               group,
		Environment:         env,
	}
}

func (e ManifestEnvironmentLoaderError) Error() string {
	return fmt.Sprintf("%s:%s:%s: %s", e.ManifestPath, e.Group, e.Environment, e.Reason)
}

type ManifestProjectLoaderError struct {
	ManifestLoaderError
	Project string
}

func newManifestProjectLoaderError(manifest string, project string, reason string) ManifestProjectLoaderError {
	return ManifestProjectLoaderError{
		ManifestLoaderError: newManifestLoaderError(manifest, reason),
		Project:             project,
	}
}

func (e ManifestProjectLoaderError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.ManifestPath, e.Project, e.Reason)
}

func LoadManifest(context *ManifestLoaderContext) (Manifest, []error) {
	manifestPath := filepath.Clean(context.ManifestPath)

	if !files.IsYamlFileExtension(manifestPath) {
		return Manifest{}, []error{newManifestLoaderError(context.ManifestPath, "manifest file is not a yaml")}
	}

	exists, err := files.DoesFileExist(context.Fs, manifestPath)

	if err != nil {
		return Manifest{}, []error{err}
	}

	if !exists {
		return Manifest{}, []error{newManifestLoaderError(context.ManifestPath, "specified manifest file is either no file or does not exist")}
	}

	data, err := afero.ReadFile(context.Fs, manifestPath)

	if err != nil {
		return Manifest{}, []error{newManifestLoaderError(context.ManifestPath, fmt.Sprintf("error while reading the manifest: %s", err))}
	}

	return parseManifest(context, data)
}

func parseManifest(context *ManifestLoaderContext, data []byte) (Manifest, []error) {
	manifestPath := filepath.Clean(context.ManifestPath)

	manifest, err := parseManifestFile(context, data)
	if err != nil {
		return Manifest{}, err
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
		errors = append(errors, newManifestLoaderError(context.ManifestPath, "no projects defined in manifest"))
	}

	environmentDefinitions, manifestErrors := toEnvironments(context, manifest.EnvironmentGroups)

	if manifestErrors != nil {
		errors = append(errors, manifestErrors...)
	} else if len(environmentDefinitions) == 0 {
		errors = append(errors, newManifestLoaderError(context.ManifestPath, "no environments defined in manifest"))
	}

	if errors != nil {
		return Manifest{}, errors
	}

	return Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
	}, nil
}

func parseManifestFile(context *ManifestLoaderContext, data []byte) (manifest, []error) {
	var m manifest
	var errs []error

	err := yaml.UnmarshalStrict(data, &m)

	if err != nil {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("error during parsing the manifest: %s", err)))
	}

	err = validateManifestVersion(m.ManifestVersion)
	if err != nil {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("invalid manifest definition: %s", err)))
	}

	if len(m.Projects) == 0 {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, "invalid manifest definition: no `projects` defined"))
	}

	if len(m.EnvironmentGroups) == 0 {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, "invalid manifest definition: no `environmentGroups` defined"))
	}

	if len(errs) != 0 {
		return manifest{}, errs
	}

	return m, nil
}

var maxSupportedManifestVersion, _ = util.ParseVersion(version.ManifestVersion)
var minSupportedManifestVersion, _ = util.ParseVersion(version.MinManifestVersion)

func validateManifestVersion(manifestVersion string) error {
	if len(manifestVersion) == 0 {
		return fmt.Errorf("`manifestVersion` missing")
	}

	v, err := util.ParseVersion(manifestVersion)
	if err != nil {
		return fmt.Errorf("invalid `manifestVersion`: %w", err)
	}

	if v.SmallerThan(minSupportedManifestVersion) {
		return fmt.Errorf("`manifestVersion` %s is no longer supported. Min required version is %s, please update manifest", manifestVersion, version.MinManifestVersion)
	}

	if v.GreaterThan(maxSupportedManifestVersion) {
		return fmt.Errorf("`manifestVersion` %s is not supported by monaco %s. Max supported version is %s, please check manifest or update monaco", manifestVersion, version.MonitoringAsCode, version.ManifestVersion)
	}

	return nil
}

func toEnvironments(context *ManifestLoaderContext, groups []group) (map[string]EnvironmentDefinition, []error) {
	var errors []error
	environments := make(map[string]EnvironmentDefinition)

	for i, group := range groups {
		if group.Name == "" {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("missing group name on index `%d`", i)))
			continue
		}

		for _, conf := range group.Environments {
			env, configErrors := toEnvironment(context, conf, group.Name)

			if configErrors != nil {
				errors = append(errors, configErrors...)
				continue
			}

			if _, found := environments[env.Name]; found {
				errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("environment with name `%s` already exists", env.Name)))
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

	if config.Url.Value == "" {
		errors = append(errors, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, "no `url` configured or value is blank"))
	}

	urlType, err := extractUrlType(config)
	if err != nil {
		errors = append(errors, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("failed to parse URL %v", err)))
	}

	if len(errors) > 0 {
		return EnvironmentDefinition{}, errors
	}

	return EnvironmentDefinition{
		Name: config.Name,
		url: UrlDefinition{
			Type:  urlType,
			Value: strings.TrimSuffix(config.Url.Value, "/"),
		},
		Token: token,
		Group: group,
	}, nil
}

func extractUrlType(config environment) (UrlType, error) {
	if config.Url.Type == "" || config.Url.Type == util.ToString(ValueUrlType) {
		return ValueUrlType, nil
	}

	if config.Url.Type == util.ToString(EnvironmentUrlType) {
		return EnvironmentUrlType, nil
	}

	return "", fmt.Errorf("%s is not a valid URL Type", config.Url.Type)
}

func parseToken(context *ManifestLoaderContext, config environment, group string, token tokenConfig) (Token, error) {
	var tokenType string

	if token.Type == "" {
		tokenType = "environment"
	} else {
		tokenType = token.Type
	}

	switch tokenType {
	case "environment":
		return parseEnvironmentToken(context, group, config, token)
	}

	return nil, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("unknwon token type `%s`", tokenType))
}

func parseEnvironmentToken(context *ManifestLoaderContext, group string, config environment, token tokenConfig) (Token, error) {
	if val, found := token.Config["name"]; found {
		return &EnvironmentVariableToken{util.ToString(val)}, nil
	}

	return nil, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, "missing key `name` in token config")
}

func toProjectDefinitions(context *projectLoaderContext, definitions []project) (map[string]ProjectDefinition, []error) {
	var errors []error
	result := make(map[string]ProjectDefinition)

	definitionErrors := checkForDuplicateDefinitions(context, definitions)
	if len(definitionErrors) > 0 {
		return nil, definitionErrors
	}

	for _, project := range definitions {
		parsed, projectErrors := parseProjectDefinition(context, project)

		if projectErrors != nil {
			errors = append(errors, projectErrors...)
			continue
		}

		for _, project := range parsed {
			if p, found := result[project.Name]; found {
				errors = append(errors, newManifestLoaderError(context.manifestPath, fmt.Sprintf("duplicated project name `%s` used by %s and %s", project.Name, p, project)))
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

func checkForDuplicateDefinitions(context *projectLoaderContext, definitions []project) (errors []error) {
	definedIds := map[string]struct{}{}
	for _, project := range definitions {
		if _, found := definedIds[project.Name]; found {
			errors = append(errors, newManifestLoaderError(context.manifestPath, fmt.Sprintf("duplicated project name `%s`", project.Name)))
		}
		definedIds[project.Name] = struct{}{}
	}
	return errors
}

func parseProjectDefinition(context *projectLoaderContext, project project) ([]ProjectDefinition, []error) {
	var projectType string

	if project.Type == "" {
		projectType = simpleProjectType
	} else {
		projectType = project.Type
	}

	switch projectType {
	case simpleProjectType:
		return parseSimpleProjectDefinition(context, project)
	case groupProjectType:
		return parseGroupingProjectDefinition(context, project)
	default:
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			fmt.Sprintf("invalid project type `%s`", projectType))}
	}
}

func parseSimpleProjectDefinition(context *projectLoaderContext, project project) ([]ProjectDefinition, []error) {
	if project.Path == "" && project.Name == "" {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			"project is missing both name and path")}
	}

	if strings.ContainsAny(project.Name, `/\`) {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			`project name is not allowed to contain '/' or '\'`)}
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
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name, fmt.Sprintf("failed to read project dir: %v", err))}
	}

	var result []ProjectDefinition

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		result = append(result, ProjectDefinition{
			Name:  project.Name + "." + file.Name(),
			Group: project.Name,
			Path:  filepath.Join(projectPath, file.Name()),
		})
	}

	if result == nil {
		// TODO should we really fail here?
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			fmt.Sprintf("no projects found in `%s`", projectPath))}
	}

	return result, nil
}
