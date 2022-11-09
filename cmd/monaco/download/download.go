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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	project "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/maps"
	"github.com/spf13/afero"
	"net/url"
	"os"
	"path"
	"strings"
)

//go:generate mockgen -source=download.go -destination=download_mock.go -package=download -write_package_comment=false Command

// Command is used to test the CLi commands properly without executing the actual monaco download.
//
// The actual implementations are in the [DefaultCommand] struct.
type Command interface {
	DownloadConfigsBasedOnManifest(fs afero.Fs, manifestFile, projectName, specificEnvironmentName, outputFolder string, apiNamesToDownload []string) error
	DownloadConfigs(fs afero.Fs, environmentUrl, projectName, envVarName, outputFolder string, apiNamesToDownload []string) error
}

// DefaultCommand is used to implement the [Command] interface.
type DefaultCommand struct{}

// make sure DefaultCommand implements the Command interface
var (
	_ Command = (*DefaultCommand)(nil)
)

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, manifestFile, projectName, specificEnvironmentName, outputFolder string, apiNamesToDownload []string) error {

	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestFile,
	})

	if errs != nil {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest '%v'", manifestFile)
	}

	env, found := man.Environments[specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment '%v' was not available in manifest '%v'", specificEnvironmentName, manifestFile)
	}

	apisToDownload, errs := getApisToDownload(apiNamesToDownload)

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load apis")
	}

	url, err := env.GetUrl()
	if err != nil {
		errs = append(errs, err)
	}

	token, err := env.GetToken()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		util.PrintErrors(errs)
		return fmt.Errorf("failed to load manifest data")
	}

	tokenEnvVar := fmt.Sprintf("TOKEN_%s", strings.ToUpper(projectName))
	if envVarToken, ok := env.Token.(*manifest.EnvironmentVariableToken); ok {
		tokenEnvVar = envVarToken.EnvironmentVariableName
	}

	return doDownload(fs, url, fmt.Sprintf("%s_%s", projectName, specificEnvironmentName), token, tokenEnvVar, outputFolder, apisToDownload, rest.NewDynatraceClient)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, environmentUrl, projectName, envVarName, outputFolder string, apiNamesToDownload []string) error {

	apis, errors := getApisToDownload(apiNamesToDownload)

	if len(errors) > 0 {
		util.PrintErrors(errors)
		return fmt.Errorf("failed to load apis")
	}

	token := os.Getenv(envVarName)
	errors = validateParameters(envVarName, environmentUrl, projectName, token)

	if len(errors) > 0 {
		util.PrintErrors(errors)

		return fmt.Errorf("not all necessary information is present to start downloading configurations")
	}

	return doDownload(fs, environmentUrl, projectName, token, envVarName, outputFolder, apis, rest.NewDynatraceClient)
}

type dynatraceClientFactory func(environmentUrl, token string) (rest.DynatraceClient, error)

func doDownload(fs afero.Fs, environmentUrl, projectName, token, tokenEnvVarName, outputFolder string, apis api.ApiMap, clientFactory dynatraceClientFactory) error {

	errors := validateOutputFolder(fs, outputFolder, projectName)
	if len(errors) > 0 {
		util.PrintErrors(errors)

		return fmt.Errorf("output folder is invalid")
	}

	log.Info("Downloading from environment '%v' into project '%v'", environmentUrl, projectName)
	log.Debug("APIS to download: \n - %v", strings.Join(maps.Keys(apis), "\n - "))

	client, err := clientFactory(environmentUrl, token)
	if err != nil {
		return fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	downloadedConfigs := downloader.DownloadAllConfigs(apis, client, projectName)

	if len(downloadedConfigs) == 0 {
		log.Info("No configs were downloaded")
		return nil
	} else {
		log.Info("Downloaded %v configs", sumConfigs(downloadedConfigs))
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs = download.ResolveDependencies(downloadedConfigs)

	err = download.WriteToDisk(fs, downloadedConfigs, projectName, tokenEnvVarName, environmentUrl, outputFolder)
	if err != nil {
		return err
	}

	log.Info("Finished download")

	return nil
}

func sumConfigs(configs project.ConfigsPerApis) int {
	sum := 0

	for _, v := range configs {
		sum += len(v)
	}

	return sum
}

// validateParameters checks that all necessary variables have been set.
func validateParameters(envVarName, environmentUrl, projectName, token string) []error {
	errors := make([]error, 0)

	if envVarName == "" {
		errors = append(errors, fmt.Errorf("token not specified"))
	} else if token == "" {
		errors = append(errors, fmt.Errorf("the content of token '%v' is not set", envVarName))
	}

	if environmentUrl == "" {
		errors = append(errors, fmt.Errorf("url not specified"))
	}

	if _, err := url.Parse(environmentUrl); err != nil {
		errors = append(errors, fmt.Errorf("url is invalid: %w", err))
	}

	if projectName == "" {
		errors = append(errors, fmt.Errorf("project not specified"))
	}

	return errors
}

func validateOutputFolder(fs afero.Fs, outputFolder, project string) []error {
	errors := make([]error, 0)

	errors = append(errors, validateFolder(fs, outputFolder)...)
	if len(errors) > 0 {
		return errors
	}
	errors = append(errors, validateFolder(fs, path.Join(outputFolder, project))...)
	return errors
}

func validateFolder(fs afero.Fs, path string) []error {
	errors := make([]error, 0)
	exists, err := afero.Exists(fs, path)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to check if output folder '%s' exists: %w", path, err))
	}
	if exists {
		isDir, err := afero.IsDir(fs, path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to check if output folder '%s' is a folder: %w", path, err))
		}
		if !isDir {
			errors = append(errors, fmt.Errorf("unable to write to '%s': file exists and is not a directory", path))
		}
	}

	return errors
}

// Get all v2 apis and filter for the selected ones
func getApisToDownload(apisToDownload []string) (api.ApiMap, []error) {

	var errors []error

	apis, unknownApis := api.NewApis().FilterApisByName(apisToDownload)
	if len(unknownApis) > 0 {
		errors = append(errors, fmt.Errorf("APIs '%v' are not known. Please consult our documentation for known API-names", strings.Join(unknownApis, ",")))
	}

	apis, filtered := apis.Filter(func(api api.Api) bool {
		return api.ShouldSkipDownload()
	})

	if len(filtered) > 0 {
		keys := strings.Join(maps.Keys(filtered), ", ")
		log.Info("APIs that won't be downloaded and need manual creation: '%v'.", keys)
	}

	if len(apis) == 0 {
		errors = append(errors, fmt.Errorf("no APIs to download"))
	}

	return apis, errors
}
