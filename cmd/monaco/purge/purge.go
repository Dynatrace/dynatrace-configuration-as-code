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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
	"golang.org/x/exp/maps"
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

	return purgeConfigs(maps.Values(mani.Environments), apis)
}

func purgeConfigs(environments []manifest.EnvironmentDefinition, apis api.APIs) error {

	for _, env := range environments {
		err := purgeForEnvironment(env, apis)
		if err != nil {
			return err
		}
	}

	return nil
}

func purgeForEnvironment(env manifest.EnvironmentDefinition, apis api.APIs) error {
	clients, err := dynatrace.CreateClientSet(env.URL.Value, env.Auth)

	if err != nil {
		return fmt.Errorf("failed to create a client for env `%s` due to the following error: %w", env.Name, err)
	}

	ctx := context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})

	log.WithCtxFields(ctx).Info("Deleting configs for environment `%s`", env.Name)

	deleteErrors := delete.AllConfigs(ctx, clients.Classic(), apis)
	deleteErrors = append(deleteErrors, delete.AllSettingsObjects(ctx, clients.Settings())...)
	deleteErrors = append(deleteErrors, delete.AllAutomations(ctx, clients.Automation())...)
	deleteErrors = append(deleteErrors, delete.AllBuckets(ctx, clients.Bucket())...)

	if len(deleteErrors) > 0 {
		log.Error("Encountered %d errors while puring configurations from environment %s, further manual cleanup may be needed. Errors:", len(deleteErrors), env.Name)
		errutils.PrintErrors(deleteErrors)
	}
	return nil
}
