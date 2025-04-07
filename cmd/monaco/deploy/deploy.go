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

	if err := validateEnvironments(ctx, loadedEnvironments, loadedManifest.Environments); err != nil {
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

	clientSets, err := dynatrace.CreateEnvironmentClients(ctx, loadedManifest.Environments, dryRun)
	if err != nil {
		formattedErr := fmt.Errorf("failed to create API clients: %w", err)
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateError, formattedErr, "", nil)
		return formattedErr
	}

	err = deploy.DeployForAllEnvironments(ctx, loadedEnvironments, clientSets, deploy.DeployConfigsOptions{ContinueOnErr: continueOnErr, DryRun: dryRun})
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
	environments, errs := project.LoadEnvironments(ctx, fs, project.ProjectLoaderContext{
		KnownApis:       api.NewAPIs().Filter(api.RemoveDisabled).GetApiNameLookup(),
		WorkingDir:      filepath.Dir(manifestPath),
		Manifest:        *man,
		ParametersSerde: config.DefaultParameterParsers,
	}, specificProjects)

	if errs != nil {
		log.Error("Failed to load environments - %d errors occurred:", len(errs))
		for _, err := range errs {
			log.WithFields(field.Error(err)).Error(err.Error())
		}
		return nil, fmt.Errorf("failed to load environments - %d errors occurred", len(errs))
	}

	return environments, nil
}

type KindCoordinates map[string][]coordinate.Coordinate
type KindCoordinatesPerEnvironment map[string]KindCoordinates
type CoordinatesPerEnvironment map[string][]coordinate.Coordinate

func validateEnvironments(ctx context.Context, environments []project.Environment, envs manifest.Environments) error {
	openPipelineKindCoordinatesPerEnvironment := KindCoordinatesPerEnvironment{}
	platformCoordinatesPerEnvironment := CoordinatesPerEnvironment{}
	for _, e := range environments {
		configs := e.AllConfigs()
		openPipelineKindCoordinates, found := openPipelineKindCoordinatesPerEnvironment[e.Name]
		if !found {
			openPipelineKindCoordinates = KindCoordinates{}
			openPipelineKindCoordinatesPerEnvironment[e.Name] = openPipelineKindCoordinates
		}
		collectOpenPipelineCoordinatesByKind(configs, openPipelineKindCoordinates)

		platformCoordinatesPerEnvironment[e.Name] = append(platformCoordinatesPerEnvironment[e.Name], collectPlatformCoordinates(configs)...)
	}

	errs := collectRequiresPlatformErrors(platformCoordinatesPerEnvironment, envs)
	errs = append(errs, collectOpenPipelineCoordinateErrors(openPipelineKindCoordinatesPerEnvironment)...)
	reporter := report.GetReporterFromContextOrDiscard(ctx)

	for _, err := range errs {
		reporter.ReportLoading(report.StateError, err, "", nil)
	}

	return errors.Join(errs...)
}

func collectOpenPipelineCoordinatesByKind(configs []config.Config, dest KindCoordinates) {
	for _, cfg := range configs {
		if cfg.Skip {
			continue
		}

		if openPipelineType, ok := cfg.Type.(config.OpenPipelineType); ok {
			dest[openPipelineType.Kind] = append(dest[openPipelineType.Kind], cfg.Coordinate)
		}
	}
}

func collectPlatformCoordinates(configs []config.Config) []coordinate.Coordinate {
	platformCoordinates := make([]coordinate.Coordinate, 0)

	for _, cfg := range configs {
		if cfg.Skip {
			continue
		}

		if configRequiresPlatform(cfg) {
			platformCoordinates = append(platformCoordinates, cfg.Coordinate)
		}
	}
	return platformCoordinates
}

// TODO: extend with segment, slo, etc.
func configRequiresPlatform(c config.Config) bool {
	switch c.Type.(type) {
	case config.AutomationType, config.BucketType, config.DocumentType, config.OpenPipelineType:
		return true
	default:
		return false
	}
}

func collectOpenPipelineCoordinateErrors(openPipelineKindCoordinatesPerEnvironment KindCoordinatesPerEnvironment) []error {
	errs := []error{}
	for envName, openPipelineKindCoordinates := range openPipelineKindCoordinatesPerEnvironment {

		// check for duplicate configurations for the same kind of openpipeline.
		for kind, coordinates := range openPipelineKindCoordinates {
			if len(coordinates) > 1 {
				errs = append(errs, fmt.Errorf("environment %q has multiple openpipeline configurations of kind %q: %s", envName, kind, coordinateSliceAsString(coordinates)))
			}
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

func collectRequiresPlatformErrors(platformCoordinatesPerEnvironment CoordinatesPerEnvironment, envs manifest.Environments) []error {
	errs := []error{}
	for envName, coordinates := range platformCoordinatesPerEnvironment {
		env, found := envs[envName]
		if !found || platformEnvironment(env) {
			continue
		}

		if len(coordinates) > 0 {
			exampleCoordinate := coordinates[0]
			errs = append(errs, fmt.Errorf("environment %q is not configured to access platform, but at least one configuration (e.g. %q) requires it", envName, exampleCoordinate))
		}
	}
	return errs
}

func platformEnvironment(e manifest.EnvironmentDefinition) bool {
	return e.Auth.OAuth != nil
}

// validateAuthenticationWithProjectConfigs validates each config entry against the manifest if required credentials are set
// it takes into consideration the project, environments and the skip parameter in each config entry
func validateAuthenticationWithProjectConfigs(environments []project.Environment, envDefinition manifest.Environments) error {
	for _, e := range environments {
		for _, conf := range e.AllConfigs() {
			if conf.Skip == true {
				continue
			}

			switch conf.Type.(type) {
			case config.ClassicApiType:
				if envDefinition[e.Name].Auth.Token == nil {
					return fmt.Errorf("API of type '%s' requires a token for environment '%s'", conf.Type, e.Name)
				}
			case config.SettingsType:
				t, ok := conf.Type.(config.SettingsType)
				if ok && t.AllUserPermission != nil && envDefinition[e.Name].Auth.OAuth == nil {
					return fmt.Errorf("using permission property on settings API requires OAuth, schema '%s' enviroment '%s'", t.SchemaId, e.Name)
				}
				if envDefinition[e.Name].Auth.Token == nil && envDefinition[e.Name].Auth.OAuth == nil {
					return fmt.Errorf("API of type '%s' requires a token or OAuth for environment '%s'", conf.Type, e.Name)
				}
			default:
				if envDefinition[e.Name].Auth.OAuth == nil {
					return fmt.Errorf("API of type '%s' requires OAuth for environment '%s'", conf.Type, e.Name)
				}
			}
		}
	}
	return nil
}
