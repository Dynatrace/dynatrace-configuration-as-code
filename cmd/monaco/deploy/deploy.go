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
	"errors"
	"fmt"
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
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"path/filepath"
	"strings"
)

func deployConfigs(fs afero.Fs, manifestPath string, environmentGroups []string, specificEnvironments []string, specificProjects []string, continueOnErr bool, dryRun bool) error {
	absManifestPath, err := absPath(manifestPath)
	if err != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %w", manifestPath, err)
	}
	loadedManifest, err := loadManifest(fs, absManifestPath, environmentGroups, specificEnvironments)
	if err != nil {
		return err
	}

	ok := verifyEnvironmentGen(loadedManifest.Environments, dryRun)
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
	}

	loadedProjects, err := loadProjects(fs, absManifestPath, loadedManifest, specificProjects)
	if err != nil {
		return err
	}

	if err := checkEnvironments(loadedProjects, loadedManifest.Environments); err != nil {
		return err
	}

	logging.LogProjectsInfo(loadedProjects)
	logging.LogEnvironmentsInfo(loadedManifest.Environments)

	clientSets, err := dynatrace.CreateEnvironmentClients(loadedManifest.Environments)
	if err != nil {
		return fmt.Errorf("failed to create API clients: %w", err)
	}

	err = deploy.Deploy(loadedProjects, clientSets, deploy.DeployConfigsOptions{ContinueOnErr: continueOnErr, DryRun: dryRun})
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

func loadManifest(fs afero.Fs, manifestPath string, groups []string, environments []string) (*manifest.Manifest, error) {
	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Groups:       groups,
		Environments: environments,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})

	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return nil, errors.New("error while loading manifest")
	}

	return &m, nil
}

func verifyEnvironmentGen(environments manifest.Environments, dryRun bool) bool {
	if !dryRun {
		return dynatrace.VerifyEnvironmentGeneration(environments)

	}
	return true
}

func loadProjects(fs afero.Fs, manifestPath string, man *manifest.Manifest, specificProjects []string) ([]project.Project, error) {
	projects, errs := project.LoadProjects(fs, project.ProjectLoaderContext{
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

func checkEnvironments(projects []project.Project, envs manifest.Environments) error {
	errs := []error{}
	openPipelineKindCoordinatesPerEnvironment := map[string]map[string][]coordinate.Coordinate{}
	for _, p := range projects {
		for envName, cfgPerType := range p.Configs {
			env, found := envs[envName]
			if !found {
				errs = append(errs, fmt.Errorf("undefined environment %q", envName))
				continue
			}

			openPipelineKindCoordinates := openPipelineKindCoordinatesPerEnvironment[envName]
			if openPipelineKindCoordinates == nil {
				openPipelineKindCoordinates = make(map[string][]coordinate.Coordinate)
			}
			cfgPerType.ForEveryConfigDo(func(cfg config.Config) {
				if cfg.Skip {
					return
				}

				if openPipelineType, ok := cfg.Type.(config.OpenPipelineType); ok {
					coordinates := openPipelineKindCoordinates[openPipelineType.Kind]
					coordinates = append(coordinates, cfg.Coordinate)
					openPipelineKindCoordinates[openPipelineType.Kind] = coordinates
				}
			})
			openPipelineKindCoordinatesPerEnvironment[envName] = openPipelineKindCoordinates

			if err := checkIfConfigsForEnvironmentRequireUnavailablePlatform(env, cfgPerType); err != nil {
				errs = append(errs, err)
			}
		}
	}

	for envName, openPipelineKindCoordinates := range openPipelineKindCoordinatesPerEnvironment {
		for kind, coordinates := range openPipelineKindCoordinates {
			if len(coordinates) > 1 {
				errs = append(errs, fmt.Errorf("environment %q has multiple OpenPipeline configurations of kind %q: %s", envName, kind, coordinateSliceAsString(coordinates)))
			}
		}
	}

	return errors.Join(errs...)
}

func coordinateSliceAsString(coordinates []coordinate.Coordinate) string {
	coordinateStrings := make([]string, 0, len(coordinates))
	for _, c := range coordinates {
		coordinateStrings = append(coordinateStrings, c.String())

	}
	return strings.Join(coordinateStrings, ", ")
}

// checkIfConfigsForEnvironmentRequireUnavailablePlatform returns an error if a config requires platform and it is not configured in the environment.
func checkIfConfigsForEnvironmentRequireUnavailablePlatform(env manifest.EnvironmentDefinition, cfgPerType v2.ConfigsPerType) error {
	if platformEnvironment(env) {
		return nil
	}

	requiresPlatform := false
	var exampleCoordinate coordinate.Coordinate
	cfgPerType.ForEveryConfigDo(func(cfg config.Config) {
		if cfg.Skip || requiresPlatform {
			return
		}

		if configRequiresPlatform(cfg) {
			exampleCoordinate = cfg.Coordinate
			requiresPlatform = true
		}
	})

	if requiresPlatform {
		return fmt.Errorf("environment %q is not configured to access platform, but at least one configuration (e.g. %q) requires it", env.Name, exampleCoordinate)
	}
	return nil
}

func platformEnvironment(e manifest.EnvironmentDefinition) bool {
	return e.Auth.OAuth != nil
}

func configRequiresPlatform(c config.Config) bool {
	switch c.Type.(type) {
	case config.AutomationType, config.BucketType, config.DocumentType, config.OpenPipelineType:
		return true
	default:
		return false
	}
}
