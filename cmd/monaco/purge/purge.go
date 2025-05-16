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
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

func purge(ctx context.Context, fs afero.Fs, deploymentManifestPath string, environmentNames []string, apiNames []string) error {

	deploymentManifestPath = filepath.Clean(deploymentManifestPath)
	deploymentManifestPath, manifestErr := filepath.Abs(deploymentManifestPath)

	if manifestErr != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %w", deploymentManifestPath, manifestErr)
	}

	apis := api.NewAPIs().Filter(api.RetainByName(apiNames), api.RemoveDisabled, api.RemoveNonDeletable)

	mani, manifestLoadError := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: deploymentManifestPath,
		Environments: environmentNames,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})

	if manifestLoadError != nil {
		errutils.PrintErrors(manifestLoadError)
		return errors.New("error while loading manifest")
	}

	return purgeConfigs(ctx, maps.Values(mani.Environments.SelectedEnvironments), apis)
}

func purgeConfigs(ctx context.Context, environments []manifest.EnvironmentDefinition, apis api.APIs) error {

	for _, env := range environments {
		err := purgeForEnvironment(ctx, env, apis)
		if err != nil {
			return err
		}
	}

	return nil
}

func purgeForEnvironment(ctx context.Context, env manifest.EnvironmentDefinition, apis api.APIs) error {
	ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})

	clients, err := client.CreateClientSet(ctx, env.URL.Value, env.Auth)
	if err != nil {
		return fmt.Errorf("failed to create a client for env `%s`: %w", env.Name, err)
	}

	log.WithCtxFields(ctx).Info("Deleting configs for environment `%s`", env.Name)

	if err := delete.All(ctx, *clients, apis); err != nil {
		log.Error("Encountered errors while puring configurations from environment %s, further manual cleanup may be needed - check logs for details.", env.Name)
	}
	return nil
}
