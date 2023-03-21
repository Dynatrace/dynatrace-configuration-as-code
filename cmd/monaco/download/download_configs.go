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

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
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

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
	})
	if len(errs) > 0 {
		err := PrintAndFormatErrors(errs, "failed to load manifest '%v'", cmdOptions.manifestFile)
		return err
	}

	env, found := m.Environments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment %q was not available in manifest %q", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	ok := cmdutils.VerifyEnvironmentGeneration(manifest.Environments{env.Name: env})
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
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
			clientProvider:          defaultDynatraceClientProvider,
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
		return PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl:          cmdOptions.environmentUrl,
			token:                   token,
			tokenEnvVarName:         cmdOptions.envVarName,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			clientProvider:          defaultDynatraceClientProvider,
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

	c, err := opts.getDynatraceClient()
	if err != nil {
		return fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	c = client.LimitClientParallelRequests(c, opts.concurrentDownloadLimit)

	if ok, unknownApis := validateSpecificAPIs(apis, opts.specificAPIs); !ok {
		errutils.PrintError(fmt.Errorf("APIs '%v' are not known. Please consult our documentation for known API-names", strings.Join(unknownApis, ",")))
		return fmt.Errorf("failed to load apis")
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)
	downloadedConfigs, err := downloadConfigs(c, apis, opts)
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

func downloadConfigs(c client.Client, apis api.APIs, opts downloadConfigsOptions) (project.ConfigsPerType, error) {
	configObjects := make(project.ConfigsPerType)

	if shouldDownloadClassicConfigs(opts) {
		classicCfgs, err := downloadClassicConfigs(c, apis, opts.specificAPIs, opts.projectName)
		if err != nil {
			return nil, err
		}
		maps.Copy(configObjects, classicCfgs)
	}

	if shouldDownloadSettings(opts) {
		settingsObjects := downloadSettings(c, opts.specificSchemas, opts.projectName)
		maps.Copy(configObjects, settingsObjects)
	}

	return configObjects, nil
}

// shouldDownloadClassicConfigs returns true unless onlySettings or specificSchemas but no specificAPIs are defined
func shouldDownloadClassicConfigs(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && (len(opts.specificSchemas) == 0 || len(opts.specificAPIs) > 0)
}

// shouldDownloadSettings returns true unless onlyAPIs or specificAPIs but no specificSchemas are defined
func shouldDownloadSettings(opts downloadConfigsOptions) bool {
	return !opts.onlyAPIs && (len(opts.specificAPIs) == 0 || len(opts.specificSchemas) > 0)
}

func downloadClassicConfigs(c client.Client, apis api.APIs, specificAPIs []string, projectName string) (project.ConfigsPerType, error) {
	apisToDownload := getApisToDownload(apis, specificAPIs)
	if len(apisToDownload) == 0 {
		return nil, fmt.Errorf("no APIs to download")
	}

	if len(specificAPIs) > 0 {
		log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
		cfgs := classic.DownloadAllConfigs(apisToDownload, c, projectName)
		return cfgs, nil
	}

	log.Debug("APIs to download: \n - %v", strings.Join(maps.Keys(apisToDownload), "\n - "))
	cfgs := classic.DownloadAllConfigs(apisToDownload, c, projectName)
	return cfgs, nil
}

func downloadSettings(c client.Client, specificSchemas []string, projectName string) project.ConfigsPerType {
	if len(specificSchemas) > 0 {
		log.Debug("Settings to download: \n - %v", strings.Join(specificSchemas, "\n - "))
		s := settings.Download(c, specificSchemas, projectName)
		return s
	}

	s := settings.DownloadAll(c, projectName)
	return s
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
