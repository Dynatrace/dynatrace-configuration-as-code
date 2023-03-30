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

package purge

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"path/filepath"
)

func purge(fs afero.Fs, deploymentManifestPath string, environmentNames []string, apiNames []string) error {

	deploymentManifestPath = filepath.Clean(deploymentManifestPath)
	deploymentManifestPath, manifestErr := filepath.Abs(deploymentManifestPath)

	if manifestErr != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %w", deploymentManifestPath, manifestErr)
	}

	apis := api.NewAPIs().Filter(api.RetainByName(apiNames))

	mani, manifestLoadError := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: deploymentManifestPath,
		Environments: environmentNames,
	})

	if manifestLoadError != nil {
		errutils.PrintErrors(manifestLoadError)
		return errors.New("error while loading manifest")
	}

	deleteErrors := purgeConfigs(maps.Values(mani.Environments), apis)

	for _, e := range deleteErrors {
		log.Error("Deletion error: %s", e)
	}
	if len(deleteErrors) > 0 {
		return fmt.Errorf("encountered %v errors during delete", len(deleteErrors))
	}
	return nil
}

func purgeConfigs(environments []manifest.EnvironmentDefinition, apis api.APIs) (errors []error) {

	for _, env := range environments {
		deleteErrors := purgeForEnvironment(env, apis)

		if deleteErrors != nil {
			errors = append(errors, deleteErrors...)
		}
	}

	return errors
}

func purgeForEnvironment(env manifest.EnvironmentDefinition, apis api.APIs) []error {
	dynatraceClient, err := cmdutils.CreateDTClient(env.URL.Value, env.Auth, false)

	if err != nil {
		return []error{
			fmt.Errorf("failed to create a client for env `%s` due to the following error: %w", env.Name, err),
		}
	}

	log.Info("Deleting configs for environment `%s`", env.Name)

	errs := delete.DeleteAllConfigs(dynatraceClient, apis)
	errs = append(errs, delete.DeleteAllSettingsObjects(dynatraceClient)...)

	return errs
}
