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

package loader

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/internal/persistence"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Context holds all information for [Load]
type Context struct {
	// Fs holds the abstraction of the file system.
	Fs afero.Fs

	// ManifestPath holds the path from where the manifest should be loaded.
	ManifestPath string

	// Environments is a filter to what environments should be loaded.
	// If it's empty, all environments are loaded.
	// If both Environments and Groups are specified, the union of both results is returned.
	//
	// If Environments contains items that do not match any environment in the specified manifest file, the loading errors.
	Environments []string

	// Groups is a filter to what environment-groups (and thus environments) should be loaded.
	// If it's empty, all environment-groups are loaded.
	// If both Environments and Groups are specified, the union of both results is returned.
	//
	// If Groups contains items that do not match any environment in the specified manifest file, the loading errors.
	Groups []string

	// Opts are Options holding optional configuration for Load
	Opts Options
}

type projectLoaderContext struct {
	fs           afero.Fs
	manifestPath string
}

// Options are optional configuration for Load
type Options struct {
	DoNotResolveEnvVars      bool
	RequireEnvironmentGroups bool
	RequireAccounts          bool
}

type ManifestLoaderError struct {
	// ManifestPath is the path of the manifest file that failed to load
	ManifestPath string `json:"manifestPath"`
	// Reason describing what went wrong
	Reason string `json:"reason"`
}

func (e ManifestLoaderError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Reason)
}

func newManifestLoaderError(path string, reason string) ManifestLoaderError {
	return ManifestLoaderError{
		ManifestPath: path,
		Reason:       reason,
	}
}

type EnvironmentDetails struct {
	Group       string `json:"group"`
	Environment string `json:"environment"`
}

type EnvironmentLoaderError struct {
	ManifestLoaderError
	// EnvironmentDetails of the environment that failed to be loaded
	EnvironmentDetails EnvironmentDetails `json:"environmentDetails"`
}

func newManifestEnvironmentLoaderError(manifest string, group string, env string, reason string) EnvironmentLoaderError {
	return EnvironmentLoaderError{
		ManifestLoaderError: newManifestLoaderError(manifest, reason),
		EnvironmentDetails: EnvironmentDetails{
			Group:       group,
			Environment: env,
		},
	}
}

func (e EnvironmentLoaderError) Error() string {
	return fmt.Sprintf("%s:%s:%s: %s", e.ManifestPath, e.EnvironmentDetails.Group, e.EnvironmentDetails.Environment, e.Reason)
}

type ProjectLoaderError struct {
	ManifestLoaderError
	// Project name that failed to be loaded
	Project string `json:"project"`
}

func newManifestProjectLoaderError(manifest string, project string, reason string) ProjectLoaderError {
	return ProjectLoaderError{
		ManifestLoaderError: newManifestLoaderError(manifest, reason),
		Project:             project,
	}
}

func (e ProjectLoaderError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.ManifestPath, e.Project, e.Reason)
}

func Load(context *Context) (manifest.Manifest, []error) {
	log.WithFields(field.F("manifestPath", context.ManifestPath)).Info("Loading manifest %q. Restrictions: groups=%q, environments=%q", context.ManifestPath, context.Groups, context.Environments)

	manifestYAML, err := readManifestYAML(context)
	if err != nil {
		return manifest.Manifest{}, []error{err}
	}

	// check that the manifestVersion is ok
	if err := validateVersion(manifestYAML); err != nil {
		return manifest.Manifest{}, []error{newManifestLoaderError(context.ManifestPath, fmt.Sprintf("invalid manifest definition: %s", err))}
	}

	if context.Opts.RequireEnvironmentGroups && len(manifestYAML.EnvironmentGroups) == 0 {
		return manifest.Manifest{}, []error{newManifestLoaderError(context.ManifestPath, "'environmentGroups' are required, but not defined")}
	}
	if context.Opts.RequireAccounts && len(manifestYAML.Accounts) == 0 {
		return manifest.Manifest{}, []error{newManifestLoaderError(context.ManifestPath, "'accounts' are required, but not defined")}
	}

	manifestPath := filepath.Clean(context.ManifestPath)

	workingDir := filepath.Dir(manifestPath)
	var workingDirFs afero.Fs

	if workingDir == "." {
		workingDirFs = context.Fs
	} else {
		workingDirFs = afero.NewBasePathFs(context.Fs, workingDir)
	}

	relativeManifestPath := filepath.Base(manifestPath)

	var errs []error

	// projects
	projectDefinitions, projectErrors := parseProjects(&projectLoaderContext{
		fs:           workingDirFs,
		manifestPath: relativeManifestPath,
	}, manifestYAML.Projects)
	if projectErrors != nil {
		errs = append(errs, projectErrors...)
	}

	// environments
	var environmentDefinitions map[string]manifest.EnvironmentDefinition
	if len(manifestYAML.EnvironmentGroups) > 0 {
		var manifestErrors []error
		if environmentDefinitions, manifestErrors = parseEnvironments(context, manifestYAML.EnvironmentGroups); manifestErrors != nil {
			errs = append(errs, manifestErrors...)
		} else if len(environmentDefinitions) == 0 {
			errs = append(errs, newManifestLoaderError(context.ManifestPath, "no environments defined in manifest"))
		}
	}

	// accounts
	accounts, accErr := parseAccounts(context, manifestYAML.Accounts)
	if accErr != nil {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, accErr.Error()))
	}

	// params
	params, paramErrs := parseParameters(context.Fs, config.DefaultParameterParsers, manifestYAML.Parameters)
	if paramErrs != nil {
		errs = append(errs, newManifestLoaderError(context.ManifestPath, paramErrs.Error()))
	}

	// if any errors occurred up to now, return them
	if errs != nil {
		return manifest.Manifest{}, errs
	}

	m := manifest.Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
		Accounts:     accounts,
	}

	if len(params) > 0 {
		m.Parameters = params
	}

	return m, nil
}

func parseAuth(context *Context, a persistence.Auth) (manifest.Auth, error) {
	var mAuth manifest.Auth

	if a.Token == nil && a.OAuth == nil {
		return manifest.Auth{}, errors.New("no token or OAuth credentials provided")
	}

	if a.Token != nil {
		token, err := parseAuthSecret(context, a.Token)
		if err != nil {
			return manifest.Auth{}, fmt.Errorf("failed to parse token: %w", err)
		}
		mAuth.Token = &token
	}

	if a.OAuth != nil {
		oauth, err := parseOAuth(context, a.OAuth)
		if err != nil {
			return manifest.Auth{}, fmt.Errorf("failed to parse OAuth credentials: %w", err)
		}
		mAuth.OAuth = oauth
	}

	return mAuth, nil
}

func parseAuthSecret(context *Context, s *persistence.AuthSecret) (manifest.AuthSecret, error) {
	if !(s.Type == persistence.TypeEnvironment || s.Type == "") {
		return manifest.AuthSecret{}, errors.New("type must be 'environment'")
	}

	if s.Name == "" {
		return manifest.AuthSecret{}, errors.New("no name given or empty")
	}

	if context.Opts.DoNotResolveEnvVars {
		log.Debug("Skipped resolving environment variable %s based on loader options", s.Name)
		return manifest.AuthSecret{
			Name:  s.Name,
			Value: secret.MaskedString(fmt.Sprintf("SKIPPED RESOLUTION OF ENV_VAR: %s", s.Name)),
		}, nil
	}

	v, f := os.LookupEnv(s.Name)
	if !f {
		return manifest.AuthSecret{}, fmt.Errorf("environment-variable %q was not found", s.Name)
	}

	if v == "" {
		return manifest.AuthSecret{}, fmt.Errorf("environment-variable %q found, but the value resolved is empty", s.Name)
	}

	return manifest.AuthSecret{Name: s.Name, Value: secret.MaskedString(v)}, nil
}

func parseOAuth(context *Context, a *persistence.OAuth) (*manifest.OAuth, error) {
	clientID, err := parseAuthSecret(context, &a.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ClientID: %w", err)
	}

	clientSecret, err := parseAuthSecret(context, &a.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ClientSecret: %w", err)
	}

	if a.TokenEndpoint != nil {
		urlDef, err := parseURLDefinition(context, *a.TokenEndpoint)
		if err != nil {
			return nil, fmt.Errorf(`failed to parse "tokenEndpoint": %w`, err)
		}

		return &manifest.OAuth{
			ClientID:      clientID,
			ClientSecret:  clientSecret,
			TokenEndpoint: &urlDef,
		}, nil
	}

	return &manifest.OAuth{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		TokenEndpoint: nil,
	}, nil
}

func readManifestYAML(context *Context) (persistence.Manifest, error) {
	manifestPath := filepath.Clean(context.ManifestPath)

	if !files.IsYamlFileExtension(manifestPath) {
		return persistence.Manifest{}, newManifestLoaderError(context.ManifestPath, "manifest file is not a yaml")
	}

	if exists, err := files.DoesFileExist(context.Fs, manifestPath); err != nil {
		return persistence.Manifest{}, err
	} else if !exists {
		return persistence.Manifest{}, newManifestLoaderError(context.ManifestPath, "manifest file does not exist")
	}

	rawData, err := afero.ReadFile(context.Fs, manifestPath)
	if err != nil {
		return persistence.Manifest{}, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("error while reading the manifest: %s", err))
	}

	var m persistence.Manifest

	err = yaml.UnmarshalStrict(rawData, &m)
	if err != nil {
		return persistence.Manifest{}, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("error during parsing the manifest: %s", err))
	}
	return m, nil
}

var maxSupportedManifestVersion, _ = version2.ParseVersion(version.ManifestVersion)
var minSupportedManifestVersion, _ = version2.ParseVersion(version.MinManifestVersion)

func validateVersion(m persistence.Manifest) error {

	if len(m.ManifestVersion) == 0 {
		return errors.New("`manifestVersion` missing")
	}

	v, err := version2.ParseVersion(m.ManifestVersion)
	if err != nil {
		return fmt.Errorf("invalid `manifestVersion`: %w", err)
	}

	if v.SmallerThan(minSupportedManifestVersion) {
		return fmt.Errorf("`manifestVersion` %s is no longer supported. Min required version is %s, please update manifest", m.ManifestVersion, version.MinManifestVersion)
	}

	if v.GreaterThan(maxSupportedManifestVersion) {
		return fmt.Errorf("`manifestVersion` %s is not supported by monaco %s. Max supported version is %s, please check manifest or update monaco", m.ManifestVersion, version.MonitoringAsCode, version.ManifestVersion)
	}

	return nil
}

func parseEnvironments(context *Context, groups []persistence.Group) (map[string]manifest.EnvironmentDefinition, []error) { // nolint:gocognit
	var errors []error
	environments := make(map[string]manifest.EnvironmentDefinition)

	groupNames := make(map[string]bool, len(groups))
	envNames := make(map[string]bool, len(groups))

	for i, group := range groups {
		if group.Name == "" {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("missing group name on index `%d`", i)))
		}

		if groupNames[group.Name] {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("duplicated group name %q", group.Name)))
		}

		groupNames[group.Name] = true

		for j, env := range group.Environments {

			if env.Name == "" {
				errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("missing environment name in group %q on index `%d`", group.Name, j)))
				continue
			}

			if envNames[env.Name] {
				errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("duplicated environment name %q", env.Name)))
				continue
			}
			envNames[env.Name] = true

			// skip loading if environments is not empty, the environments does not contain the env name, or the group should not be included
			if shouldSkipEnv(context, group, env) {
				log.WithFields(field.F("manifestPath", context.ManifestPath)).Debug("skipping loading of environment %q", env.Name)
				continue
			}

			parsedEnv, configErrors := parseSingleEnvironment(context, env, group.Name)

			if configErrors != nil {
				errors = append(errors, configErrors...)
				continue
			}

			environments[parsedEnv.Name] = parsedEnv
		}
	}

	// validate that all required groups & environments are included
	for _, g := range context.Groups {
		if !groupNames[g] {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("requested group %q not found", g)))
		}
	}

	for _, e := range context.Environments {
		if !envNames[e] {
			errors = append(errors, newManifestLoaderError(context.ManifestPath, fmt.Sprintf("requested environment %q not found", e)))
		}
	}

	if errors != nil {
		return nil, errors
	}

	return environments, nil
}

func shouldSkipEnv(context *Context, group persistence.Group, env persistence.Environment) bool {
	// if nothing is restricted, everything is allowed
	if len(context.Groups) == 0 && len(context.Environments) == 0 {
		return false
	}

	if slices.Contains(context.Groups, group.Name) {
		return false
	}

	if slices.Contains(context.Environments, env.Name) {
		return false
	}

	return true
}

func parseSingleEnvironment(context *Context, config persistence.Environment, group string) (manifest.EnvironmentDefinition, []error) {
	var errs []error

	a, err := parseAuth(context, config.Auth)
	if err != nil {
		errs = append(errs, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, fmt.Sprintf("failed to parse auth section: %s", err)))
	}

	urlDef, err := parseURLDefinition(context, config.URL)
	if err != nil {
		errs = append(errs, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, err.Error()))
	}

	if len(errs) > 0 {
		return manifest.EnvironmentDefinition{}, errs
	}

	return manifest.EnvironmentDefinition{
		Name:  config.Name,
		URL:   urlDef,
		Auth:  a,
		Group: group,
	}, nil
}

func parseURLDefinition(context *Context, u persistence.TypedValue) (manifest.URLDefinition, error) {

	// Depending on the type, the url.value either contains the env var name or the direct value of the url
	if u.Value == "" {
		return manifest.URLDefinition{}, errors.New("no `Url` configured or value is blank")
	}

	if u.Type == "" || u.Type == persistence.TypeValue {
		val := strings.TrimSuffix(u.Value, "/")

		return manifest.URLDefinition{
			Type:  manifest.ValueURLType,
			Value: val,
		}, nil
	}

	if u.Type == persistence.TypeEnvironment {

		if context.Opts.DoNotResolveEnvVars {
			log.Debug("Skipped resolving environment variable %s based on loader options", u.Value)
			return manifest.URLDefinition{
				Type:  manifest.EnvironmentURLType,
				Value: fmt.Sprintf("SKIPPED RESOLUTION OF ENV_VAR: %s", u.Value),
				Name:  u.Value,
			}, nil
		}

		val, found := os.LookupEnv(u.Value)
		if !found {
			return manifest.URLDefinition{}, fmt.Errorf("environment variable %q could not be found", u.Value)
		}

		if val == "" {
			return manifest.URLDefinition{}, fmt.Errorf("environment variable %q is defined but has no value", u.Value)
		}

		val = strings.TrimSuffix(val, "/")

		return manifest.URLDefinition{
			Type:  manifest.EnvironmentURLType,
			Value: val,
			Name:  u.Value,
		}, nil

	}

	return manifest.URLDefinition{}, fmt.Errorf("%q is not a valid URL type", u.Type)
}

func parseProjects(context *projectLoaderContext, definitions []persistence.Project) (map[string]manifest.ProjectDefinition, []error) {
	var errors []error
	result := make(map[string]manifest.ProjectDefinition)

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

func checkForDuplicateDefinitions(context *projectLoaderContext, definitions []persistence.Project) (errors []error) {
	definedIds := map[string]struct{}{}
	for _, project := range definitions {
		if _, found := definedIds[project.Name]; found {
			errors = append(errors, newManifestLoaderError(context.manifestPath, fmt.Sprintf("duplicated project name `%s`", project.Name)))
		}
		definedIds[project.Name] = struct{}{}
	}
	return errors
}

func parseProjectDefinition(context *projectLoaderContext, project persistence.Project) ([]manifest.ProjectDefinition, []error) {
	var projectType string

	if project.Type == "" {
		projectType = persistence.SimpleProjectType
	} else {
		projectType = project.Type
	}

	if project.Name == "" {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name, "project name is required")}
	}

	switch projectType {
	case persistence.SimpleProjectType:
		return parseSimpleProjectDefinition(context, project)
	case persistence.GroupProjectType:
		return parseGroupingProjectDefinition(context, project)
	default:
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			fmt.Sprintf("invalid project type `%s`", projectType))}
	}
}

func parseSimpleProjectDefinition(context *projectLoaderContext, project persistence.Project) ([]manifest.ProjectDefinition, []error) {
	if project.Path == "" && project.Name == "" {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			"project is missing both name and path")}
	}

	if strings.ContainsAny(project.Name, `/\`) {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name,
			`project name is not allowed to contain '/' or '\'`)}
	}

	if project.Path == "" {
		return []manifest.ProjectDefinition{
			{
				Name: project.Name,
				Path: project.Name,
			},
		}, nil
	}

	return []manifest.ProjectDefinition{
		{
			Name: project.Name,
			Path: project.Path,
		},
	}, nil
}

func parseGroupingProjectDefinition(context *projectLoaderContext, project persistence.Project) ([]manifest.ProjectDefinition, []error) {
	projectPath := filepath.FromSlash(project.Path)

	files, err := afero.ReadDir(context.fs, projectPath)

	if err != nil {
		return nil, []error{newManifestProjectLoaderError(context.manifestPath, project.Name, fmt.Sprintf("failed to read project dir: %v", err))}
	}

	var result []manifest.ProjectDefinition

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		result = append(result, manifest.ProjectDefinition{
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
