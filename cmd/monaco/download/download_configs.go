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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/dependency_resolution"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
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

func (d DefaultCommand) DownloadConfigsBasedOnManifest(fs afero.Fs, cmdOptions downloadCmdOptions) error {

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
		Environments: []string{cmdOptions.specificEnvironmentName},
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

	downloaders, err := makeDownloaders(options)
	if err != nil {
		return err
	}
	return doDownloadConfigs(fs, downloaders, options)
}

func (d DefaultCommand) DownloadConfigs(fs afero.Fs, cmdOptions downloadCmdOptions) error {
	a, errors := cmdOptions.auth.mapToAuth()
	errors = append(errors, validateParameters(cmdOptions.environmentURL, cmdOptions.projectName)...)

	if len(errors) > 0 {
		return printAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
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

	downloaders, err := makeDownloaders(options)
	if err != nil {
		return err
	}

	return doDownloadConfigs(fs, downloaders, options)
}

type downloadConfigsOptions struct {
	downloadOptionsShared
	specificAPIs    []string
	specificSchemas []string
	onlyAPIs        bool
	onlySettings    bool
	onlyAutomation  bool
}

func doDownloadConfigs(fs afero.Fs, downloaders downloaders, opts downloadConfigsOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentURL, opts.projectName)
	downloadedConfigs, err := downloadConfigs(downloaders, opts)
	if err != nil {
		return err
	}

	log.Info("Resolving dependencies between configurations")
	downloadedConfigs = dependency_resolution.ResolveDependencies(downloadedConfigs)

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

func downloadConfigs(downloaders downloaders, opts downloadConfigsOptions) (project.ConfigsPerType, error) {
	configs := make(project.ConfigsPerType)

	if shouldDownloadClassicConfigs(opts) {
		classicAPIs := makeClassicAPIs(opts.specificAPIs)
		classicCfgs, err := downloaders.Classic().Download(opts.projectName, classicAPIs...)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, classicCfgs)
	}

	if shouldDownloadSettings(opts) {
		settingTypes := makeSettingTypes(opts.specificSchemas)
		settingCfgs, err := downloaders.Settings().Download(opts.projectName, settingTypes...)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, settingCfgs)
	}

	if shouldDownloadAutomationResources(opts) {
		automationCfgs, err := downloaders.Automation().Download(opts.projectName)
		if err != nil {
			return nil, err
		}
		copyConfigs(configs, automationCfgs)
	}

	return configs, nil
}

func makeClassicAPIs(specificAPIs []string) []v2.ClassicApiType {
	var classicAPIs []v2.ClassicApiType
	for _, api := range specificAPIs {
		classicAPIs = append(classicAPIs, v2.ClassicApiType{Api: api})
	}
	return classicAPIs
}

func makeSettingTypes(specificSchemas []string) []v2.SettingsType {
	var settingTypes []v2.SettingsType
	for _, schema := range specificSchemas {
		settingTypes = append(settingTypes, v2.SettingsType{SchemaId: schema})
	}
	return settingTypes
}

func copyConfigs(dest, src project.ConfigsPerType) {
	for k, v := range src {
		dest[k] = v
	}
}

func shouldDownloadClassicConfigs(opts downloadConfigsOptions) bool {
	return !opts.onlyAutomation && !opts.onlySettings &&
		(len(opts.specificSchemas) == 0 || len(opts.specificAPIs) > 0)
}

func shouldDownloadSettings(opts downloadConfigsOptions) bool {
	return !opts.onlyAutomation && !opts.onlyAPIs &&
		(len(opts.specificAPIs) == 0 || len(opts.specificSchemas) > 0)
}

func shouldDownloadAutomationResources(opts downloadConfigsOptions) bool {
	return !opts.onlySettings && !opts.onlyAPIs &&
		(len(opts.specificAPIs) == 0 || len(opts.specificSchemas) > 0) &&
		(len(opts.specificSchemas) == 0 || len(opts.specificAPIs) > 0) &&
		featureflags.AutomationResources().Enabled()
}
