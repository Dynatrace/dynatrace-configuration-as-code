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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"os"
	"path/filepath"
	"strings"

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

var maxSupportedManifestVersion, _ = version2.ParseVersion(version.ManifestVersion)
var minSupportedManifestVersion, _ = version2.ParseVersion(version.MinManifestVersion)

func validateManifestVersion(manifestVersion string) error {
	if len(manifestVersion) == 0 {
		return fmt.Errorf("`manifestVersion` missing")
	}

	v, err := version2.ParseVersion(manifestVersion)
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

	groupNames := make(map[string]bool, len(groups))

	for i, group := range groups {
		if group.Name == "" {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("missing group name on index `%d`", i)))
		}

		if groupNames[group.Name] {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("duplicated group name %q", group.Name)))
		}

		groupNames[group.Name] = true

		for _, conf := range group.Environments {
			env, configErrors := toEnvironment(context, conf, group.Name)

			if configErrors != nil {
				errors = append(errors, configErrors...)
				continue
			}

			if _, found := environments[env.Name]; found {
				errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("duplicated environment name `%s`", env.Name)))
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

	envType, err := parseEnvironmentType(context, config, group)
	if err != nil {
		errors = append(errors, err)
	}

	token, err := parseToken(context, config, group, config.Token)

	if err != nil {
		errors = append(errors, err)
	}

	urlDef, err := parseUrlDefinition(context, config, group)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return EnvironmentDefinition{}, errors
	}

	return EnvironmentDefinition{
		Name:  config.Name,
		Type:  envType,
		url:   urlDef,
		Token: token,
		Group: group,
	}, nil
}

func parseUrlDefinition(context *ManifestLoaderContext, config environment, group string) (UrlDefinition, error) {

	// Depending on the type, the url.value either contains the env var name or the direct value of the url
	if config.Url.Value == "" {
		return UrlDefinition{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, "no `url` configured or value is blank")
	}

	if config.Url.Type == "" || config.Url.Type == string(ValueUrlType) {
		return UrlDefinition{
			Type:  ValueUrlType,
			Value: config.Url.Value,
		}, nil
	}

	if config.Url.Type == string(EnvironmentUrlType) {
		val, found := os.LookupEnv(config.Url.Value)
		if !found {
			return UrlDefinition{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("environment variable %q could not be found", config.Url.Value))
		}

		if val == "" {
			return UrlDefinition{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("environment variable %q is defined but has no value", config.Url.Value))
		}

		return UrlDefinition{
			Type:  EnvironmentUrlType,
			Value: val,
			Name:  config.Url.Value,
		}, nil

	}

	return UrlDefinition{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("%q is not a valid URL type", config.Url.Type))
}

func parseEnvironmentType(context *ManifestLoaderContext, config environment, g string) (EnvironmentType, error) {
	switch strings.ToLower(config.Type) {
	case "":
		fallthrough
	case "classic":
		return Classic, nil
	case "platform":
		return Platform, nil
	}

	return Classic, newManifestEnvironmentLoaderError(context.ManifestPath, g, config.Name, fmt.Sprintf(`invalid environment-type %q. Allowed values are "classic" (default) and "platform"`, config.Type))
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

	return Token{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("unknown token type `%s`", tokenType))
}

func parseEnvironmentToken(context *ManifestLoaderContext, group string, config environment, token tokenConfig) (Token, error) {

	if token.Name == "" {
		return Token{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, "token `name` is missing or empty")
	}

	// resolve token value immediately
	val, found := os.LookupEnv(token.Name)
	if !found {
		return Token{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("no environment variable found for token %q", token.Name))
	}

	if val == "" {
		return Token{}, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("environment variable for token %q is empty", token.Name))
	}

	return Token{Name: token.Name, Value: val}, nil
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
