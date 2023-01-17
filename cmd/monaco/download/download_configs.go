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
	"os"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

type downloadCommandOptions struct {
	downloadCommandOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
}

type manifestDownloadOptions struct {
	manifestFile            string
	specificEnvironmentName string
	downloadCommandOptions
}

type directDownloadOptions struct {
	environmentUrl, envVarName string
	downloadCommandOptions
}

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions manifestDownloadOptions) error {
	envUrl, token, tokenEnvVar, err := getEnvFromManifest(fs, cmdOptions.manifestFile, cmdOptions.specificEnvironmentName, cmdOptions.projectName)
	if err != nil {
		return err
	}
	printUploadToSameEnvironmentWarning(envUrl, token)

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := rest.ConcurrentRequestLimitFromEnv(true)

	options := downloadOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl:          envUrl,
			token:                   token,
			tokenEnvVarName:         tokenEnvVar,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			clientProvider:          client.NewDynatraceClient,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
	}
	return doDownloadConfigs(fs, api.NewApis(), options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions directDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := rest.ConcurrentRequestLimitFromEnv(true)
	errors := validateParameters(cmdOptions.envVarName, cmdOptions.environmentUrl, cmdOptions.projectName, token)

	if len(errors) > 0 {
		return PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl:          cmdOptions.environmentUrl,
			token:                   token,
			tokenEnvVarName:         cmdOptions.envVarName,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			clientProvider:          client.NewDynatraceClient,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
	}
	return doDownloadConfigs(fs, api.NewApis(), options)
}

type downloadOptions struct {
	downloadOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
}

func doDownloadConfigs(fs afero.Fs, apis api.ApiMap, opts downloadOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	if ok, unknownApis := validateSpecificAPIs(apis, opts.specificAPIs); !ok {
		errutils.PrintError(fmt.Errorf("APIs '%v' are not known. Please consult our documentation for known API-names", strings.Join(unknownApis, ",")))
		return fmt.Errorf("failed to load apis")
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)
	downloadedConfigs, err := downloadConfigs(apis, opts)
	if err != nil {
		return err
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs = download.ResolveDependencies(downloadedConfigs)

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, err, fs)
}

func validateSpecificAPIs(a api.ApiMap, apiNames []string) (valid bool, unknownAPIs []string) {
	for _, v := range apiNames {
		if !a.Contains(v) {
			unknownAPIs = append(unknownAPIs, v)
		}
	}
	return len(unknownAPIs) == 0, unknownAPIs
}

func downloadConfigs(apis api.ApiMap, opts downloadOptions) (project.ConfigsPerType, error) {
	c, err := opts.getDynatraceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	c = client.LimitClientParallelRequests(c, opts.concurrentDownloadLimit)

	apisToDownload := getApisToDownload(apis, opts.specificAPIs)
	if len(apisToDownload) == 0 {
		return nil, fmt.Errorf("no APIs to download")
	}

	configObjects := make(project.ConfigsPerType)

	// download specific APIs only
	if len(opts.specificAPIs) > 0 {
		log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
		c := classic.DownloadAllConfigs(apisToDownload, c, opts.projectName)
		maps.Copy(configObjects, c)
	}

	// download specific settings only
	if len(opts.specificSchemas) > 0 {
		log.Debug("Settings to download: \n - %v", strings.Join(opts.specificSchemas, "\n - "))
		s := settings.Download(c, opts.specificSchemas, opts.projectName)
		maps.Copy(configObjects, s)
	}

	// return specific download objects
	if len(opts.specificSchemas) > 0 || len(opts.specificAPIs) > 0 {
		return configObjects, nil
	}

	// if nothing was specified specifically, lets download all configs and settings
	if !opts.onlySettings {
		log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
		configObjects = classic.DownloadAllConfigs(apisToDownload, c, opts.projectName)
	}
	if !opts.onlyAPIs {
		settingsObjects := settings.DownloadAll(c, opts.projectName)
		maps.Copy(configObjects, settingsObjects)
	}
	return configObjects, nil
}

// Get all v2 apis and filter for the selected ones
func getApisToDownload(apis api.ApiMap, specificAPIs []string) api.ApiMap {
	if len(specificAPIs) > 0 {
		return apis.Filter(api.RetainByName(specificAPIs), skipDownloadFilter)
	} else {
		return apis.Filter(skipDownloadFilter, removeDeprecatedEndpoints)
	}
}

func skipDownloadFilter(api api.Api) bool {
	if api.ShouldSkipDownload() {
		log.Info("API can not be downloaded and needs manual creation: '%v'.", api.GetId())
		return true
	}
	return false
}

func removeDeprecatedEndpoints(api api.Api) bool {
	if api.DeprecatedBy() != "" {
		log.Warn("API %q is deprecated by %q and will not be downloaded", api.GetId(), api.DeprecatedBy())
		return true
	}
	return false
}
