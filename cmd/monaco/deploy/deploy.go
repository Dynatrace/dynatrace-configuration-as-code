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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
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
	for _, p := range projects {
		for envName, cfgPerType := range p.Configs {
			if _, found := envs[envName]; !found {
				return fmt.Errorf("cannot find environment `%s`", envName)
			}
			for _, cfgs := range cfgPerType {
				if err := checkConfigsForEnvironment(envs[envName], cfgs); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func checkConfigsForEnvironment(env manifest.EnvironmentDefinition, cfgs []config.Config) error {
	for i := range cfgs {
		if !cfgs[i].Skip && onlyAvailableOnPlatform(&cfgs[i]) && !platformEnvironment(env) {
			return fmt.Errorf("enviroment %q is not specified as platform, but at least one of configurations (e.g. %q) is platform exclusive", env.Name, cfgs[i].Coordinate)
		}
	}
	return nil
}

func platformEnvironment(e manifest.EnvironmentDefinition) bool {
	return e.Auth.OAuth != nil
}

func onlyAvailableOnPlatform(c *config.Config) bool {
	if _, ok := c.Type.(config.AutomationType); ok {
		return true
	}
	_, ok := c.Type.(config.BucketType)
	return ok
}
