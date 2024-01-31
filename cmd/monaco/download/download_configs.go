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
	"errors"
	"fmt"
	automationClient "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	bucketClient "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/secret"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/dependency_resolution"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/id_extraction"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	projectv2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"os"
)

type downloadCmdOptions struct {
	sharedDownloadCmdOptions
	environmentURL string
	auth
	manifestFile            string
	specificEnvironmentName string
	specificAPIs            []string
	specificSchemas         []string
	onlyAPIs                bool
	onlySettings            bool
	onlyAutomation          bool
}

type auth struct {
	token, clientID, clientSecret string
}

func (a auth) mapToAuth() (*manifest.Auth, []error) {
	errs := make([]error, 0)
	retVal := manifest.Auth{}

	if v, err := readEnvVariable(a.token); err != nil {
		errs = append(errs, err)
	} else {
		retVal.Token = v
	}

	if a.clientID != "" && a.clientSecret != "" {
		retVal.OAuth = &manifest.OAuth{}
		if v, err := readEnvVariable(a.clientID); err != nil {
			errs = append(errs, err)
		} else {
			retVal.OAuth.ClientID = v
		}
		if v, err := readEnvVariable(a.clientSecret); err != nil {
			errs = append(errs, err)
		} else {
			retVal.OAuth.ClientSecret = v
		}
	}
	return &retVal, errs
}

func readEnvVariable(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: secret.MaskedString(content)}, nil
}

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions downloadCmdOptions) error {

	m, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
		Environments: []string{cmdOptions.specificEnvironmentName},
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	if len(errs) > 0 {
		err := printAndFormatErrors(errs, "failed to load manifest '%v'", cmdOptions.manifestFile)
		return err
	}

	env, found := m.Environments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment %q was not available in manifest %q", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	ok := dynatrace.VerifyEnvironmentGeneration(manifest.Environments{env.Name: env})
	if !ok {
		return fmt.Errorf("unable to verify Dynatrace environment generation")
	}

	printUploadToSameEnvironmentWarning(env)

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL:         env.URL.Value,
			auth:                   env.Auth,
			outputFolder:           cmdOptions.outputFolder,
			projectName:            cmdOptions.projectName,
			forceOverwriteManifest: cmdOptions.forceOverwrite,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
		onlyAutomation:  cmdOptions.onlyAutomation,
	}

	if errs := options.valid(); len(errs) != 0 {
		err := printAndFormatErrors(errs, "command options are not valid")
		return err
	}

	clientSet, err := dynatrace.CreateClientSet(options.environmentURL, options.auth)
	if err != nil {
		return err
	}

	return doDownloadConfigs(fs, clientSet, prepareAPIs(api.NewAPIs(), options), options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions downloadCmdOptions) error {
	a, errs := cmdOptions.auth.mapToAuth()
	errs = append(errs, validateParameters(cmdOptions.environmentURL, cmdOptions.projectName)...)

	if len(errs) > 0 {
		return printAndFormatErrors(errs, "not all necessary information is present to start downloading configurations")
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL:         cmdOptions.environmentURL,
			auth:                   *a,
			outputFolder:           cmdOptions.outputFolder,
			projectName:            cmdOptions.projectName,
			forceOverwriteManifest: cmdOptions.forceOverwrite,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
		onlyAutomation:  cmdOptions.onlyAutomation,
	}

	if errs := options.valid(); len(errs) != 0 {
		err := printAndFormatErrors(errs, "command options are not valid")
		return err
	}

	clientSet, err := dynatrace.CreateClientSet(options.environmentURL, options.auth)
	if err != nil {
		return err
	}

	return doDownloadConfigs(fs, clientSet, prepareAPIs(api.NewAPIs(), options), options)
}

type downloadConfigsOptions struct {
	downloadOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
	onlyAutomation  bool
}

func (opts downloadConfigsOptions) valid() []error {
	var retVal []error
	knownEndpoints := api.NewAPIs()
	for _, e := range opts.specificAPIs {
		if !knownEndpoints.Contains(e) {
			retVal = append(retVal, fmt.Errorf("unknown (or unsupported) classic endpoint with name %q provided via \"--api\" flag. A list of supported classic endpoints is in the documentation", e))
		}
	}

	return retVal
}

func doDownloadConfigs(fs afero.Fs, clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentURL, opts.projectName)
	downloadedConfigs, err := downloadConfigs(clientSet, apisToDownload, opts, defaultDownloadFn)
	if err != nil {
		return err
	}

	if len(downloadedConfigs) == 0 {
		log.Info("No configurations downloaded. No project will be created.")
		return nil
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs, err = dependency_resolution.ResolveDependencies(downloadedConfigs)
	if err != nil {
		return err
	}

	log.Info("Extracting additional identifiers into YAML parameters")
	// must happen after dep-resolution, as it removes IDs from the JSONs in which the dep-resolution searches as well
	downloadedConfigs, err = id_extraction.ExtractIDsIntoYAML(downloadedConfigs)
	if err != nil {
		return err
	}

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

type downloadFn struct {
	classicDownload    func(dtclient.Client, string, api.APIs, classic.ContentFilters) (projectv2.ConfigsPerType, error)
	settingsDownload   func(dtclient.SettingsClient, string, settings.Filters, ...config.SettingsType) (projectv2.ConfigsPerType, error)
	automationDownload func(*automationClient.Client, string, ...config.AutomationType) (projectv2.ConfigsPerType, error)
	bucketDownload     func(*bucketClient.Client, string) (projectv2.ConfigsPerType, error)
}

var defaultDownloadFn = downloadFn{
	classicDownload:    classic.Download,
	settingsDownload:   settings.Download,
	automationDownload: automation.Download,
	bucketDownload:     bucket.Download,
}

func downloadConfigs(clientSet *client.ClientSet, apisToDownload api.APIs, opts downloadConfigsOptions, fn downloadFn) (project.ConfigsPerType, error) {
	configs := make(project.ConfigsPerType)

	if shouldDownloadConfigs(opts) {
		classicCfgs, err := fn.classicDownload(clientSet.Classic(), opts.projectName, prepareAPIs(apisToDownload, opts), classic.ApiContentFilters)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, classicCfgs)
	}

	if shouldDownloadSettings(opts) {
		log.Info("Downloading settings objects")
		settingCfgs, err := fn.settingsDownload(clientSet.Settings(), opts.projectName, settings.DefaultSettingsFilters, makeSettingTypes(opts.specificSchemas)...)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, settingCfgs)
	}

	if shouldDownloadAutomationResources(opts) {
		if opts.auth.OAuth != nil {
			log.Info("Downloading automation resources")
			automationCfgs, err := fn.automationDownload(clientSet.Automation(), opts.projectName)
			if err != nil {
				return nil, err
			}
			copyConfigs(configs, automationCfgs)
		} else if opts.onlyAutomation {
			return nil, errors.New("can't download automation resources: no OAuth credentials configured")
		}
	}

	if shouldDownloadBuckets(opts) && opts.auth.OAuth != nil {
		log.Info("Downloading Grail buckets")
		bucketCfgs, err := fn.bucketDownload(clientSet.Bucket(), opts.projectName)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, bucketCfgs)
	}

	return configs, nil
}

func makeSettingTypes(specificSchemas []string) []config.SettingsType {
	var settingTypes []config.SettingsType
	for _, schema := range specificSchemas {
		settingTypes = append(settingTypes, config.SettingsType{SchemaId: schema})
	}
	return settingTypes
}

func copyConfigs(dest, src project.ConfigsPerType) {
	for k, v := range src {
		dest[k] = v
	}
}

// shouldDownloadConfigs returns true unless onlySettings or specificSchemas but no specificAPIs are defined
func shouldDownloadConfigs(opts downloadConfigsOptions) bool {
	return !opts.onlyAutomation && !opts.onlySettings && (len(opts.specificSchemas) == 0 || len(opts.specificAPIs) > 0)
}

// shouldDownloadSettings returns true unless onlyAPIs or specificAPIs but no specificSchemas are defined
func shouldDownloadSettings(opts downloadConfigsOptions) bool {
	return !opts.onlyAutomation && !opts.onlyAPIs && (len(opts.specificAPIs) == 0 || len(opts.specificSchemas) > 0)
}

// shouldDownloadAutomationResources returns true unless download is limited to settings or config API types
func shouldDownloadAutomationResources(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && len(opts.specificSchemas) == 0 &&
		!opts.onlyAPIs && len(opts.specificAPIs) == 0
}

// shouldDownloadBuckets returns true if download is not limited to another specific type
func shouldDownloadBuckets(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && len(opts.specificSchemas) == 0 && // only settings requested
		!opts.onlyAPIs && len(opts.specificAPIs) == 0 && // only Config APIs requested
		!opts.onlyAutomation
}
