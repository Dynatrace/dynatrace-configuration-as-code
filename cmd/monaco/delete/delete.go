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

package delete

import (
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	"golang.org/x/exp/maps"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
)

func Delete(fs afero.Fs, deploymentManifestPath string, deleteFile string, environmentNames []string, environmentGroups []string) error {

	deploymentManifestPath = filepath.Clean(deploymentManifestPath)
	deploymentManifestPath, manifestErr := filepath.Abs(deploymentManifestPath)

	if manifestErr != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %w", deploymentManifestPath, manifestErr)
	}

	apis := api.NewAPIs()

	manifest, manifestLoadError := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: deploymentManifestPath,
		Environments: environmentNames,
		Groups:       environmentGroups,
	})

	if manifestLoadError != nil {
		errutils.PrintErrors(manifestLoadError)
		return errors.New("error while loading manifest")
	}

	entriesToDelete, errs := delete.LoadEntriesToDelete(fs, apis.GetNames(), deleteFile)
	if errs != nil {
		return fmt.Errorf("encountered errors while parsing delete.yaml: %s", errs)
	}

	deleteErrors := deleteConfigs(maps.Values(manifest.Environments), apis, entriesToDelete)

	for _, e := range deleteErrors {
		log.WithFields(field.Error(e)).Error("Deletion error: %s", e)
	}
	if len(deleteErrors) > 0 {
		return fmt.Errorf("encountered %v errors during delete", len(deleteErrors))
	}
	return nil
}

func deleteConfigs(environments []manifest.EnvironmentDefinition, apis api.APIs, entriesToDelete map[string][]delete.DeletePointer) (errors []error) {

	for _, env := range environments {
		deleteErrors := deleteConfigForEnvironment(env, apis, entriesToDelete)

		if deleteErrors != nil {
			errors = append(errors, deleteErrors...)
		}
	}

	return errors
}

func deleteConfigForEnvironment(env manifest.EnvironmentDefinition, apis api.APIs, entriesToDelete map[string][]delete.DeletePointer) []error {

	ctx := context.WithValue(context.TODO(), log.CtxKeyEnv{}, log.CtxValEnv{Name: env.Name, Group: env.Group})

	automationResources := map[string]config.AutomationResource{
		string(config.Workflow):         config.Workflow,
		string(config.BusinessCalendar): config.BusinessCalendar,
		string(config.SchedulingRule):   config.SchedulingRule,
	}

	platformTypes := maps.Keys(automationResources)
	platformTypes = append(platformTypes, "bucket")

	if env.Auth.OAuth == nil && containsPlatformTypes(entriesToDelete, platformTypes) {
		log.WithCtxFields(ctx).Warn("Delete file contains Dynatrace Platform specific types, but no oAuth credentials are defined for environment %q - Dynatrace Platform configurations won't be deleted.", env.Name)
	}

	clientSet, err := dynatrace.CreateClientSet(env.URL.Value, env.Auth)
	if err != nil {
		return []error{
			fmt.Errorf("failed to create API client for environment %q due to the following error: %w", env.Name, err),
		}
	}

	log.WithCtxFields(ctx).Info("Deleting configs for environment %q...", env.Name)

	return delete.Configs(
		ctx,
		delete.ClientSet{
			Classic:    clientSet.Classic(),
			Settings:   clientSet.Settings(),
			Automation: clientSet.Automation(),
		},
		apis,
		automationResources,
		entriesToDelete)
}

func containsPlatformTypes(entriesToDelete map[string][]delete.DeletePointer, platformTypes []string) bool {
	for _, t := range platformTypes {
		if _, contains := entriesToDelete[t]; contains {
			return true
		}
	}
	return false
}
