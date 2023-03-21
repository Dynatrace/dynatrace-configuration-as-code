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
	"net/http"
	"os"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
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

	env, err := cmdutils.GetEnvFromManifest(fs, cmdOptions.manifestFile, cmdOptions.specificEnvironmentName)
	if err != nil {
		return err
	}

	printUploadToSameEnvironmentWarning(env)

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl:          env.URL.Value,
			token:                   env.Auth.Token.Value,
			tokenEnvVarName:         env.Auth.Token.Name,
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
	return doDownloadConfigs(fs, api.NewAPIs(), options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions directDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	errors := validateParameters(cmdOptions.envVarName, cmdOptions.environmentUrl, cmdOptions.projectName, token)

	if len(errors) > 0 {
		return errutils.PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadConfigsOptions{
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
	return doDownloadConfigs(fs, api.NewAPIs(), options)
}

type downloadConfigsOptions struct {
	downloadOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
}

func doDownloadConfigs(fs afero.Fs, apis api.APIs, opts downloadConfigsOptions) error {
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

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

func validateSpecificAPIs(a api.APIs, apiNames []string) (valid bool, unknownAPIs []string) {
	for _, v := range apiNames {
		if !a.Contains(v) {
			unknownAPIs = append(unknownAPIs, v)
		}
	}
	return len(unknownAPIs) == 0, unknownAPIs
}

func downloadConfigs(apis api.APIs, opts downloadConfigsOptions) (project.ConfigsPerType, error) {
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
func getApisToDownload(apis api.APIs, specificAPIs []string) api.APIs {
	if len(specificAPIs) > 0 {
		return apis.Filter(api.RetainByName(specificAPIs), skipDownloadFilter)
	} else {
		return apis.Filter(skipDownloadFilter, removeDeprecatedEndpoints)
	}
}

func skipDownloadFilter(api api.API) bool {
	if api.SkipDownload {
		log.Info("API can not be downloaded and needs manual creation: '%v'.", api.ID)
		return true
	}
	return false
}

func removeDeprecatedEndpoints(api api.API) bool {
	if api.DeprecatedBy != "" {
		log.Warn("API %q is deprecated by %q and will not be downloaded", api.ID, api.DeprecatedBy)
		return true
	}
	return false
}

// printUploadToSameEnvironmentWarning function may display a warning message on the console,
// notifying the user that downloaded objects cannot be uploaded to the same environment.
// It verifies the version of the tenant and, depending on the result, it may or may not display the warning.
func printUploadToSameEnvironmentWarning(env manifest.EnvironmentDefinition) {
	var serverVersion version.Version
	var err error

	var httpClient *http.Client
	if env.Type == manifest.Classic {
		httpClient = client.NewTokenAuthClient(env.Auth.Token.Value)
	} else {
		credentials := client.OauthCredentials{
			ClientID:     env.Auth.OAuth.ClientID.Value,
			ClientSecret: env.Auth.OAuth.ClientSecret.Value,
			TokenURL:     env.Auth.OAuth.TokenEndpoint.Value,
		}
		httpClient = client.NewOAuthClient(credentials)
	}

	serverVersion, err = client.GetDynatraceVersion(httpClient, env.URL.Value)
	if err != nil {
		log.Error("Unable to determine server version %q: %w", env.URL.Value, err)
		return
	}
	if serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262}) {
		logUploadToSameEnvironmentWarning()
	}
}
