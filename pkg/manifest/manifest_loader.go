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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/slices"
	version2 "github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/oauth2/endpoints"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"strings"
)

type ManifestLoaderContext struct {
	Fs           afero.Fs
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
}

type projectLoaderContext struct {
	fs           afero.Fs
	manifestPath string
}

type manifestLoaderError struct {
	ManifestPath string
	Reason       string
}

func (e manifestLoaderError) Error() string {
	return fmt.Sprintf("%s: %s", e.ManifestPath, e.Reason)
}

type environmentLoaderError struct {
	manifestLoaderError
	Group       string
	Environment string
}

func newManifestEnvironmentLoaderError(manifest string, group string, env string, reason string) environmentLoaderError {
	return environmentLoaderError{
		manifestLoaderError: manifestLoaderError{manifest, reason},
		Group:               group,
		Environment:         env,
	}
}

func (e environmentLoaderError) Error() string {
	return fmt.Sprintf("%s:%s:%s: %s", e.ManifestPath, e.Group, e.Environment, e.Reason)
}

type projectLoaderError struct {
	manifestLoaderError
	Project string
}

func newManifestProjectLoaderError(manifest string, project string, reason string) projectLoaderError {
	return projectLoaderError{
		manifestLoaderError: manifestLoaderError{manifest, reason},
		Project:             project,
	}
}

func (e projectLoaderError) Error() string {
	return fmt.Sprintf("%s:%s: %s", e.ManifestPath, e.Project, e.Reason)
}

func LoadManifest(context *ManifestLoaderContext) (Manifest, []error) {
	log.Debug("Loading manifest %q. Restrictions: groups=%q, environments=%q", context.ManifestPath, context.Groups, context.Environments)

	manifestYAML, err := readManifestYAML(context)
	if err != nil {
		return Manifest{}, []error{err}
	}
	if errs := verifyManifestYAML(manifestYAML); errs != nil {
		var retErrs []error
		for _, e := range errs {
			retErrs = append(retErrs, manifestLoaderError{context.ManifestPath, fmt.Sprintf("invalid manifest definition: %s", e)})
		}
		return Manifest{}, retErrs
	}

	manifestPath := filepath.Clean(context.ManifestPath)
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
	}, manifestYAML.Projects)

	if projectErrors != nil {
		errors = append(errors, projectErrors...)
	} else if len(projectDefinitions) == 0 {
		errors = append(errors, manifestLoaderError{context.ManifestPath, "no projects defined in manifest"})
	}

	environmentDefinitions, manifestErrors := toEnvironments(context, manifestYAML.EnvironmentGroups)

	if manifestErrors != nil {
		errors = append(errors, manifestErrors...)
	} else if len(environmentDefinitions) == 0 {
		errors = append(errors, manifestLoaderError{context.ManifestPath, "no environments defined in manifest"})
	}

	if errors != nil {
		return Manifest{}, errors
	}

	return Manifest{
		Projects:     projectDefinitions,
		Environments: environmentDefinitions,
	}, nil
}

func parseAuth(t EnvironmentType, a auth) (Auth, error) {
	token, err := parseAuthSecret(a.Token)
	if err != nil {
		return Auth{}, fmt.Errorf("error parsing token: %w", err)
	}

	if t == Classic {
		if a.OAuth != nil {
			return Auth{}, errors.New("found OAuth credentials on a Dynatrace Classic environment. If the environment is a Dynatrace Platform environment, change the type to 'Platform'")
		}

		return Auth{
			Token: token,
		}, nil
	}

	//  Platform
	if a.OAuth == nil {
		return Auth{}, errors.New("type is 'Platform', but no OAuth credentials defined")
	}

	o, err := parseOAuth(*a.OAuth)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to parse OAuth credentials: %w", err)
	}

	return Auth{
		Token: token,
		OAuth: o,
	}, nil

}

func parseAuthSecret(s authSecret) (AuthSecret, error) {

	if !(s.Type == typeEnvironment || s.Type == "") {
		return AuthSecret{}, errors.New("type must be 'environment'")
	}

	if s.Name == "" {
		return AuthSecret{}, errors.New("no name given or empty")
	}

	v, f := os.LookupEnv(s.Name)
	if !f {
		return AuthSecret{}, fmt.Errorf("environment-variable %q was not found", s.Name)
	}

	if v == "" {
		return AuthSecret{}, fmt.Errorf("environment-variable %q found, but the value resolved is empty", s.Name)
	}

	return AuthSecret{Name: s.Name, Value: v}, nil
}

func parseOAuth(a oAuth) (OAuth, error) {
	clientID, err := parseAuthSecret(a.ClientID)
	if err != nil {
		return OAuth{}, fmt.Errorf("failed to parse ClientID: %w", err)
	}

	clientSecret, err := parseAuthSecret(a.ClientSecret)
	if err != nil {
		return OAuth{}, fmt.Errorf("failed to parse ClientSecret: %w", err)
	}

	var urlDef UrlDefinition
	if a.TokenEndpoint == nil {
		urlDef = UrlDefinition{
			Value: endpoints.Dynatrace.TokenURL,
			Type:  Absent,
		}
	} else if urlDef, err = parseUrlDefinition(*a.TokenEndpoint); err != nil {
		return OAuth{}, fmt.Errorf("failed to parse \"tokenEndpoint\": %w", err)
	}

	return OAuth{
		ClientId:      clientID,
		ClientSecret:  clientSecret,
		TokenEndpoint: urlDef,
	}, nil
}

func readManifestYAML(context *ManifestLoaderContext) (manifest, error) {
	manifestPath := filepath.Clean(context.ManifestPath)

	if !files.IsYamlFileExtension(manifestPath) {
		return manifest{}, manifestLoaderError{context.ManifestPath, "manifest file is not a yaml"}
	}

	if exists, err := files.DoesFileExist(context.Fs, manifestPath); err != nil {
		return manifest{}, err
	} else if !exists {
		return manifest{}, manifestLoaderError{context.ManifestPath, "specified manifest file is either no file or does not exist"}
	}

	rawData, err := afero.ReadFile(context.Fs, manifestPath)
	if err != nil {
		return manifest{}, manifestLoaderError{context.ManifestPath, fmt.Sprintf("error while reading the manifest: %s", err)}
	}

	var m manifest

	err = yaml.UnmarshalStrict(rawData, &m)
	if err != nil {
		return manifest{}, manifestLoaderError{context.ManifestPath, fmt.Sprintf("error during parsing the manifest: %s", err)}
	}
	return m, nil
}

func verifyManifestYAML(m manifest) []error {
	var errs []error

	if err := validateManifestVersion(m.ManifestVersion); err != nil {
		errs = append(errs, err)
	}

	if len(m.Projects) == 0 { //this should be checked over the Manifest
		errs = append(errs, fmt.Errorf("no `projects` defined"))
	}

	if len(m.EnvironmentGroups) == 0 { //this should be checked over the Manifest
		errs = append(errs, fmt.Errorf("no `environmentGroups` defined"))
	}

	return errs
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
	envNames := make(map[string]bool, len(groups))

	for i, group := range groups {
		if group.Name == "" {
			errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("missing group name on index `%d`", i)})
		}

		if groupNames[group.Name] {
			errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("duplicated group name %q", group.Name)})
		}

		groupNames[group.Name] = true

		for j, env := range group.Environments {

			if env.Name == "" {
				errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("missing environment name in group %q on index `%d`", group.Name, j)})
				continue
			}

			if envNames[env.Name] {
				errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("duplicated environment name %q", env.Name)})
				continue
			}
			envNames[env.Name] = true

			// skip loading if environments is not empty, the environments does not contain the env name, or the group should not be included
			if shouldSkipEnv(context, group, env) {
				log.Debug("skipping loading of environment %q", env.Name)
				continue
			}

			parsedEnv, configErrors := parseEnvironment(context, env, group.Name)

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
			errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("requested group %q not found", g)})
		}
	}

	for _, e := range context.Environments {
		if !envNames[e] {
			errors = append(errors, manifestLoaderError{context.ManifestPath, fmt.Sprintf("requested environment %q not found", e)})
		}
	}

	if errors != nil {
		return nil, errors
	}

	return environments, nil
}

func shouldSkipEnv(context *ManifestLoaderContext, group group, env environment) bool {
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

func parseEnvironment(context *ManifestLoaderContext, config environment, group string) (EnvironmentDefinition, []error) {
	var errors []error

	envType, err := parseEnvironmentType(context, config, group)
	if err != nil {
		errors = append(errors, err)
	}

	a, err := parseCredentials(config, envType)
	if err != nil {
		errors = append(errors, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, err.Error()))
	}

	urlDef, err := parseUrlDefinition(config.Url)
	if err != nil {
		errors = append(errors, newManifestEnvironmentLoaderError(context.ManifestPath, group, config.Name, err.Error()))
	}

	if len(errors) > 0 {
		return EnvironmentDefinition{}, errors
	}

	return EnvironmentDefinition{
		Name:  config.Name,
		Type:  envType,
		Url:   urlDef,
		Auth:  a,
		Group: group,
	}, nil
}

func parseCredentials(config environment, envType EnvironmentType) (Auth, error) {
	if config.Token == nil && config.Auth == nil {
		return Auth{}, errors.New("'auth' property missing")
	}

	if config.Token != nil && config.Auth != nil {
		return Auth{}, errors.New("both 'auth' and 'token' are present")
	}

	if config.Token != nil {
		log.Warn("Environment %s: Field 'token' is deprecated, use 'auth.token' instead.", config.Name)
		token, err := parseAuthSecret(*config.Token)
		if err != nil {
			return Auth{}, fmt.Errorf("failed to parse token: %w", err)
		}

		return Auth{Token: token}, nil
	}

	a, err := parseAuth(envType, *config.Auth)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to parse auth section: %w", err)
	}

	return a, nil
}

func parseUrlDefinition(u url) (UrlDefinition, error) {

	// Depending on the type, the url.value either contains the env var name or the direct value of the url
	if u.Value == "" {
		return UrlDefinition{}, errors.New("no `Url` configured or value is blank")
	}

	if u.Type == "" || u.Type == urlTypeValue {
		return UrlDefinition{
			Type:  ValueUrlType,
			Value: u.Value,
		}, nil
	}

	if u.Type == urlTypeEnvironment {
		val, found := os.LookupEnv(u.Value)
		if !found {
			return UrlDefinition{}, fmt.Errorf("environment variable %q could not be found", u.Value)
		}

		if val == "" {
			return UrlDefinition{}, fmt.Errorf("environment variable %q is defined but has no value", u.Value)
		}

		return UrlDefinition{
			Type:  EnvironmentUrlType,
			Value: val,
			Name:  u.Value,
		}, nil

	}

	return UrlDefinition{}, fmt.Errorf("%q is not a valid URL type", u.Type)
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
				errors = append(errors, manifestLoaderError{context.manifestPath, fmt.Sprintf("duplicated project name `%s` used by %s and %s", project.Name, p, project)})
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
			errors = append(errors, manifestLoaderError{context.manifestPath, fmt.Sprintf("duplicated project name `%s`", project.Name)})
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
