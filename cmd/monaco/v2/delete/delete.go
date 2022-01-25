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
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	configDelete "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/delete/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/client"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
)

func Delete(fs afero.Fs, deploymentManifestPath string, deletePath string) error {

	deploymentManifestPath = filepath.Clean(deploymentManifestPath)
	deploymentManifestPath, manifestErr := filepath.Abs(deploymentManifestPath)
	deletePath = filepath.Clean(deletePath)
	deletePath, deleteErr := filepath.Abs(deletePath)
	deleteFileWorkingDir := strings.ReplaceAll(deletePath, "delete.yaml", "")
	deleteFile := "delete.yaml"

	if manifestErr != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %s", deploymentManifestPath, manifestErr)
	}

	if deleteErr != nil {
		return fmt.Errorf("error while finding absolute path for `%s`: %s", deletePath, deleteErr)
	}

	apis := api.NewApis()

	manifest, manifestLoadError := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: deploymentManifestPath,
	})

	if manifestLoadError != nil {
		util.PrintErrors(manifestLoadError)
		return errors.New("error while loading environments")
	}

	entriesToDelete, errors := configDelete.LoadEntriesToDelete(fs, getApiNames(apis), deleteFileWorkingDir, deleteFile)

	if errors != nil {
		return fmt.Errorf("LoadEntriesToDelte throw an error: `%s`", errors)
	}

	environments := manifest.Environments
	var result []error
	for _, env := range environments {
		client, err := createClient(env, false)

		if err != nil {
			log.Error("It was not possible to create a client for env `%s` to the following error: %s", env, err)
		}

		errs := configDelete.DeleteConfigs(client, apis, entriesToDelete)

		if errs != nil {
			log.Error("%s", errs)
		}
	}
	for _, e := range result {
		log.Error("Deletion error: %s", e)
	}
	return nil
}

func createClient(environment manifest.EnvironmentDefinition, dryRun bool) (rest.DynatraceClient, error) {
	if dryRun {
		return &client.DummyClient{}, nil
	}

	token, err := environment.GetToken()

	if err != nil {
		return nil, err
	}

	url, err := environment.GetUrl()
	if err != nil {
		return nil, err
	}

	return rest.NewDynatraceClient(url, token)
}

func getApiNames(apis map[string]api.Api) []string {
	result := make([]string, 0, len(apis))

	for api := range apis {
		result = append(result, api)
	}

	return result
}
