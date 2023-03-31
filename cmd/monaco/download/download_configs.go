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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/classic"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/settings"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"os"
	"strings"
)

type downloadCmdOptions struct {
	sharedDownloadCmdOptions
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
}

type manifestDownloadOptions struct {
	manifestFile            string
	specificEnvironmentName string
	downloadCmdOptions
}

type auth struct {
	token, clientID, clientSecret string
}

func (a auth) mapToAuth() (*manifest.Auth, []error) {
	errors := make([]error, 0)
	retVal := manifest.Auth{}

	if v, err := readEnvVariable(a.token); err != nil {
		errors = append(errors, err)
	} else {
		retVal.Token = v
	}

	if a.clientID != "" && a.clientSecret != "" {
		retVal.OAuth = &manifest.OAuth{}
		if v, err := readEnvVariable(a.clientID); err != nil {
			errors = append(errors, err)
		} else {
			retVal.OAuth.ClientID = v
		}
		if v, err := readEnvVariable(a.clientSecret); err != nil {
			errors = append(errors, err)
		} else {
			retVal.OAuth.ClientSecret = v
		}
	}
	return &retVal, errors
}

func readEnvVariable(envVar string) (manifest.AuthSecret, error) {
	var content string
	if envVar == "" {
		return manifest.AuthSecret{}, fmt.Errorf("unknown environment variable name")
	} else if content = os.Getenv(envVar); content == "" {
		return manifest.AuthSecret{}, fmt.Errorf("the content of the environment variable %q is not set", envVar)
	}
	return manifest.AuthSecret{Name: envVar, Value: content}, nil
}

type directDownloadCmdOptions struct {
	environmentURL string
	auth
	downloadCmdOptions
}

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions manifestDownloadOptions) error {

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
	})
	if len(errs) > 0 {
		err := printAndFormatErrors(errs, "failed to load manifest '%v'", cmdOptions.manifestFile)
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
			environmentURL:          env.URL.Value,
			auth:                    env.Auth,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
	}

	dtClient, err := cmdutils.CreateDTClient(env.URL.Value, env.Auth, false)
	if err != nil {
		return err
	}

	return doDownloadConfigs(fs, dtClient, api.NewAPIs(), options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions directDownloadCmdOptions) error {
	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	a, errors := cmdOptions.auth.mapToAuth()
	errors = append(errors, validateParameters(cmdOptions.environmentURL, cmdOptions.projectName)...)

	if len(errors) > 0 {
		return printAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadConfigsOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentURL:          cmdOptions.environmentURL,
			auth:                    *a,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificAPIs:    cmdOptions.specificAPIs,
		specificSchemas: cmdOptions.specificSchemas,
		onlyAPIs:        cmdOptions.onlyAPIs,
		onlySettings:    cmdOptions.onlySettings,
	}

	dtClient, err := cmdutils.CreateDTClient(options.environmentURL, options.auth, false)
	if err != nil {
		return err
	}

	return doDownloadConfigs(fs, dtClient, api.NewAPIs(), options)
}

type downloadConfigsOptions struct {
	downloadOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
}

func doDownloadConfigs(fs afero.Fs, c client.Client, apis api.APIs, opts downloadConfigsOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	c = client.LimitClientParallelRequests(c, opts.concurrentDownloadLimit)

	if ok, unknownApis := validateSpecificAPIs(apis, opts.specificAPIs); !ok {
		err := fmt.Errorf("requested APIs '%v' are not known", strings.Join(unknownApis, ","))
		log.Error("%v. Please consult our documentation for known API names.", err)
		return err
	}

	if ok, unknownSchemas := validateSpecificSchemas(c, opts.specificSchemas); !ok {
		err := fmt.Errorf("requested settings-schema(s) '%v' are not known", strings.Join(unknownSchemas, ","))
		log.Error("%v. Please consult the documentation for available schemas and verify they are available in your environment.", err)
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentURL, opts.projectName)
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

func validateSpecificSchemas(c client.SettingsClient, schemas []string) (valid bool, unknownSchemas []string) {
	if len(schemas) == 0 {
		return true, nil
	}

	schemaList, err := c.ListSchemas()
	if err != nil {
		log.Error("failed to query available Settings Schemas: %v", err)
		return false, schemas
	}
	knownSchemas := make(map[string]struct{}, len(schemaList))
	for _, s := range schemaList {
		knownSchemas[s.SchemaId] = struct{}{}
	}

	for _, s := range schemas {
		if _, exists := knownSchemas[s]; !exists {
			unknownSchemas = append(unknownSchemas, s)
		}
	}
	return len(unknownSchemas) == 0, unknownSchemas
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
	}
	return apis.Filter(skipDownloadFilter, removeDeprecatedEndpoints)
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
