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

package download

import (
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/downloader"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/spf13/afero"
	"os"
	"strings"
)

func DownloadConfigs(fs afero.Fs, workingdir, environmentUrl, projectName, envVarName string, apiNamesToDownload []string) error {

	// Initial checks ang logging basic information
	token, apis, errors := validateVariables(envVarName, environmentUrl, projectName, apiNamesToDownload)

	if len(errors) > 0 {
		util.PrintErrors(errors)

		return fmt.Errorf("not all necessary information is present to start downloading configurations")
	}

	client, err := rest.NewDynatraceClient(environmentUrl, token)
	if err != nil {
		return fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	downloadedConfigs := downloader.DownloadAllConfigs(apis, client, projectName)

	if len(downloadedConfigs) == 0 {
		log.Info("No configs were downloaded")
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs = download.ResolveDependencies(downloadedConfigs)

	err = download.WriteToDisk(fs, downloadedConfigs, projectName)
	if err != nil {
		return err
	}

	log.Info("Finished download")

	return nil
}

// validateVariables checks that all necessary variables have been set.
// If all variables have been set, a message is logged for basic information
func validateVariables(envVarName, environmentUrl, projectName string, apisToDownload []string) (string, api.ApiMap, []error) {
	token := os.Getenv(envVarName)
	errors := make([]error, 0)

	if envVarName == "" {
		errors = append(errors, fmt.Errorf("environment-variable '%v' not specified", envVarName))
	}
	if token == "" {
		errors = append(errors, fmt.Errorf("environment-variable '%v' is not set", envVarName))
	}
	if environmentUrl == "" {
		errors = append(errors, fmt.Errorf("environment-url '%v' is empty", environmentUrl))
	}
	if projectName == "" {
		errors = append(errors, fmt.Errorf("project=name '%v' is empty", environmentUrl))
	}

	// Get all v2 apis and filter for the selected ones
	apis, unknownApis := api.NewApis().FilterApisByName(apisToDownload)
	if len(unknownApis) > 0 {
		errors = append(errors, fmt.Errorf("APIs '%v' are not known. Please consult our documentation for known API-names", strings.Join(unknownApis, ",")))
	}

	apis, filtered := apis.Filter(func(api api.Api) bool {
		return api.ShouldSkipDownload()
	})

	if len(filtered) > 0 {
		keys := strings.Join(maps.Keys(filtered), ", ")
		log.Debug("Skipping APIs for download: '%v'", keys)
	}

	if len(apis) == 0 {
		errors = append(errors, fmt.Errorf("no APIs to download"))
	}

	if len(errors) > 0 {
		return "", nil, errors
	}

	if len(apisToDownload) == 0 {
		log.Info("Downloading all APIs from environment '%v' into project '%v' using environment-variable '%v'", environmentUrl, projectName, envVarName)
	} else {
		log.Info("Downloading APIs '%v' from environment '%v' into project '%v' using environment-variable '%v'", strings.Join(apisToDownload, ", "), environmentUrl, projectName, envVarName)
	}

	return token, apis, nil
}
