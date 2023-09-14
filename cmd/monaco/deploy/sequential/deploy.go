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

package sequential

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy/internal/clientset"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy/internal/logging"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/sequential"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
)

// Deploy configurations sequentially
// Deprecated: Sequential deployment is deprecated and only used if featureflags.DependencyGraphBasedDeploy is manually disabled.
func Deploy(filteredProjects []project.Project, loadedManifest *manifest.Manifest, continueOnErr bool, dryRun bool) error {
	var deployErrs []error
	sortedConfigs, err := sortConfigs(filteredProjects, loadedManifest.Environments.Names())
	if err != nil {
		return fmt.Errorf("error during configuration sort: %w", err)
	}

	for envName, cfgs := range sortedConfigs {
		env := loadedManifest.Environments[envName]
		errs := deployOnEnvironment(env, cfgs, continueOnErr, dryRun)
		deployErrs = append(deployErrs, errs...)
		if len(errs) > 0 && !continueOnErr {
			break
		}
	}

	if len(deployErrs) > 0 {
		printErrorReport(deployErrs)
		return fmt.Errorf("errors during %s", logging.GetOperationNounForLogging(dryRun))
	}
	return nil
}

func sortConfigs(projects []project.Project, environmentNames []string) (project.ConfigsPerEnvironment, error) {
	sortedConfigs, errs := sort.ConfigsPerEnvironment(projects, environmentNames)
	if errs != nil {
		errutils.PrintErrors(errs)
		return nil, errors.New("error during sort")
	}
	return sortedConfigs, nil
}

func deployOnEnvironment(env manifest.EnvironmentDefinition, cfgs []config.Config, continueOnErr bool, dryRun bool) []error {
	logging.LogDeploymentInfo(dryRun, env.Name)

	clientSet, err := clientset.NewClientSet(env, dryRun)
	if err != nil {
		return []error{fmt.Errorf("failed to create clients for envrionment %q: %w", env.Name, err)}
	}

	errs := sequential.DeployConfigs(clientSet, api.NewAPIs(), cfgs, deploy.DeployConfigsOptions{
		ContinueOnErr: continueOnErr,
		DryRun:        dryRun,
	})
	return errs
}
