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

package deploy

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy/internal/logging"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

func deployConfigs(ctx context.Context, fs afero.Fs, manifestPath string, environmentGroups []string, specificEnvironments []string, specificProjects []string, continueOnErr bool, dryRun bool) error {
	absManifestPath, err := absPath(manifestPath)
	if err != nil {
		formattedErr := fmt.Errorf("error while finding absolute path for `%s`: %w", manifestPath, err)
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateError, formattedErr, "", nil)
		return formattedErr
	}

	loadedManifest, err := loadManifest(ctx, fs, absManifestPath, environmentGroups, specificEnvironments)
	if err != nil {
		return err
	}

	ok := verifyEnvironmentGen(ctx, loadedManifest.Environments, dryRun)
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
	}

	loadedEnvironments, err := loadEnvironments(ctx, fs, absManifestPath, loadedManifest, specificProjects)
	if err != nil {
		return err
	}

	if err := validateProjectsWithEnvironments(ctx, loadedEnvironments, loadedManifest.Environments); err != nil {
		return err
	}

	logging.LogProjectsInfo(loadedEnvironments)
	logging.LogEnvironmentsInfo(loadedManifest.Environments)

	err = validateAuthenticationWithProjectConfigs(loadedEnvironments, loadedManifest.Environments)
	if err != nil {
		formattedErr := fmt.Errorf("manifest auth field misconfigured: %w", err)
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateError, formattedErr, "", nil)
		return formattedErr
	}

	for _, ev := range loadedEnvironments {
		if clientSet, err := dynatrace.CreateEnvironmentClient(ctx, loadedManifest.Environments[ev.Name], dryRun); err != nil {
			formattedErr := fmt.Errorf("failed to create API clients: %w", err)
			report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateError, formattedErr, "", nil)
			return formattedErr
		} else {
			ev.ClientSet = clientSet
		}
	}

	err = deploy.DeployForAllEnvironments(ctx, loadedEnvironments, deploy.DeployConfigsOptions{ContinueOnErr: continueOnErr, DryRun: dryRun})
	if err != nil {
		return fmt.Errorf("%v failed - check logs for details: %w", logging.GetOperationNounForLogging(dryRun), err)
	}

	log.Info("%s finished without errors", logging.GetOperationNounForLogging(dryRun))
	return nil
}

func absPath(manifestPath string) (string, error) {
	manifestPath = filepath.Clean(manifestPath)
	return filepath.Abs(manifestPath)
}

func loadManifest(ctx context.Context, fs afero.Fs, manifestPath string, groups []string, environments []string) (*manifest.Manifest, error) {
	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Groups:       groups,
		Environments: environments,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		reporter := report.GetReporterFromContextOrDiscard(ctx)
		for _, err := range errs {
			reporter.ReportLoading(report.StateError, err, "", nil)
		}
		return nil, errors.New("error while loading manifest")
	}

	return &m, nil
}

func verifyEnvironmentGen(ctx context.Context, environments manifest.Environments, dryRun bool) bool {
	if !dryRun {
		return dynatrace.VerifyEnvironmentGeneration(ctx, environments)

	}
	return true
}

func loadEnvironments(ctx context.Context, fs afero.Fs, manifestPath string, man *manifest.Manifest, specificProjects []string) ([]project.Environment, error) {
	projects, errs := project.LoadEnvironments(ctx, fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().Filter(api.RemoveDisabled).GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        *man,
		ParametersSerde: config.DefaultParameterParsers,
	}, specificProjects)

	if errs != nil {
		log.Error("Failed to load projects - %d errors occurred:", len(errs))
		for _, err := range errs {
			log.WithFields(field.Error(err)).Error(err.Error())
		}
		return nil, fmt.Errorf("failed to load projects - %d errors occurred", len(errs))
	}

	return projects, nil
}

type KindCoordinates map[string][]coordinate.Coordinate
type KindCoordinatesPerEnvironment map[string]KindCoordinates
type CoordinatesPerEnvironment map[string][]coordinate.Coordinate

func validateProjectsWithEnvironments(ctx context.Context, environments []project.Environment, envs manifest.Environments) error {
	errs := make([]error, 0)
	for _, env := range environments {
		configs := env.AllConfigs()

		openPipelineKindCoordinates := collectOpenPipelineCoordinatesByKind(configs)
		platformCoordinatesPerEnvironment := collectPlatformCoordinates(configs)
		targetEnv := getDefinitionFromConfig(env, envs)

		errs = append(errs, collectRequiresPlatformErrors(platformCoordinatesPerEnvironment, targetEnv)...)
		errs = append(errs, collectOpenPipelineCoordinateErrors(openPipelineKindCoordinates, env.Name)...)
	}

	reporter := report.GetReporterFromContextOrDiscard(ctx)

	for _, err := range errs {
		reporter.ReportLoading(report.StateError, err, "", nil)
	}

	return errors.Join(errs...)
}

func getDefinitionFromConfig(env project.Environment, envs manifest.Environments) manifest.EnvironmentDefinition {
	for _, eCfg := range envs {
		if eCfg.Name == env.Name {
			return eCfg
		}
	}
	return manifest.EnvironmentDefinition{}
}

func collectOpenPipelineCoordinatesByKind(cfgPerType []config.Config) KindCoordinates {
	dest := make(KindCoordinates)
	for _, cfg := range cfgPerType {
		if cfg.Skip {
			continue
		}

		if openPipelineType, ok := cfg.Type.(config.OpenPipelineType); ok {
			dest[openPipelineType.Kind] = append(dest[openPipelineType.Kind], cfg.Coordinate)
		}
	}
	return dest
}

func collectPlatformCoordinates(cfgPerType []config.Config) []coordinate.Coordinate {
	plaformCoordinates := []coordinate.Coordinate{}

	for _, cfg := range cfgPerType {
		if cfg.Skip {
			continue
		}

		if configRequiresPlatform(cfg) {
			plaformCoordinates = append(plaformCoordinates, cfg.Coordinate)
		}
	}
	return plaformCoordinates
}

func configRequiresPlatform(c config.Config) bool {
	switch c.Type.(type) {
	case config.AutomationType, config.BucketType, config.DocumentType, config.OpenPipelineType:
		return true
	default:
		return false
	}
}

func collectOpenPipelineCoordinateErrors(openPipelineKindCoordinates KindCoordinates, envName string) []error {
	errs := []error{}
	// check for duplicate configurations for the same kind of openpipeline.
	for kind, coordinates := range openPipelineKindCoordinates {
		if len(coordinates) > 1 {
			errs = append(errs, fmt.Errorf("environment %q has multiple openpipeline configurations of kind %q: %s", envName, kind, coordinateSliceAsString(coordinates)))
		}
	}
	return errs
}

func coordinateSliceAsString(coordinates []coordinate.Coordinate) string {
	coordinateStrings := make([]string, 0, len(coordinates))
	for _, c := range coordinates {
		coordinateStrings = append(coordinateStrings, c.String())
	}
	return strings.Join(coordinateStrings, ", ")
}

func collectRequiresPlatformErrors(coordinates []coordinate.Coordinate, env manifest.EnvironmentDefinition) []error {
	errs := make([]error, 0)

	if platformEnvironment(env) {
		return nil
	}

	if len(coordinates) > 0 {
		exampleCoordinate := coordinates[0]
		errs = append(errs, fmt.Errorf("environment %q is not configured to access platform, but at least one configuration (e.g. %q) requires it", env.Name, exampleCoordinate))
	}
	return errs
}

func platformEnvironment(e manifest.EnvironmentDefinition) bool {
	return e.Auth.OAuth != nil
}

// validateAuthenticationWithProjectConfigs validates each config entry against the manifest if required credentials are set
// it takes into consideration the project, environments and the skip parameter in each config entry
func validateAuthenticationWithProjectConfigs(environments []project.Environment, environmentConfig manifest.Environments) error {
	for _, env := range environments {
		for _, conf := range env.AllConfigs() {
			if conf.Skip == true {
				continue
			}

			switch conf.Type.(type) {
			case config.ClassicApiType:
				if environmentConfig[env.Name].Auth.Token == nil {
					return fmt.Errorf("API of type '%s' requires a token for environment '%s'", conf.Type, env.Name)
				}
			case config.SettingsType:
				t, ok := conf.Type.(config.SettingsType)
				if ok && t.AllUserPermission != nil && environmentConfig[env.Name].Auth.OAuth == nil {
					return fmt.Errorf("using permission property on settings API requires OAuth, schema '%s' enviroment '%s'", t.SchemaId, env.Name)
				}
				if environmentConfig[env.Name].Auth.Token == nil && environmentConfig[env.Name].Auth.OAuth == nil {
					return fmt.Errorf("API of type '%s' requires a token or OAuth for environment '%s'", conf.Type, env.Name)
				}
			default:
				if environmentConfig[env.Name].Auth.OAuth == nil {
					return fmt.Errorf("API of type '%s' requires OAuth for environment '%s'", conf.Type, env.Name)
				}
			}
		}
	}
	return nil
}
