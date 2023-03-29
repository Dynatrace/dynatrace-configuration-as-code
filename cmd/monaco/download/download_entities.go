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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
	"os"
	"strings"
)

type entitiesDownloadCommandOptions struct {
	sharedDownloadCmdOptions
	specificEntitiesTypes []string
}

type entitiesManifestDownloadOptions struct {
	manifestFile            string
	specificEnvironmentName string
	entitiesDownloadCommandOptions
}

type entitiesDirectDownloadOptions struct {
	environmentUrl, envVarName string
	entitiesDownloadCommandOptions
}

type downloadEntitiesOptions struct {
	downloadOptionsShared
	specificEntitiesTypes []string
}

func (d DefaultCommand) DownloadEntitiesBasedOnManifest(fs afero.Fs, cmdOptions entitiesManifestDownloadOptions) error {

	m, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: cmdOptions.manifestFile,
	})
	if len(errs) > 0 {
		err := PrintAndFormatErrors(errs, "failed to load manifest '%q'", cmdOptions.manifestFile)
		return err
	}

	env, found := m.Environments[cmdOptions.specificEnvironmentName]
	if !found {
		return fmt.Errorf("environment %q was not available in manifest %q", cmdOptions.specificEnvironmentName, cmdOptions.manifestFile)
	}

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	options := downloadEntitiesOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl:          env.URL.Value,
			auth:                    env.Auth,
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificEntitiesTypes: cmdOptions.specificEntitiesTypes,
	}

	dtClient, err := cmdutils.CreateDTClient(env, false)
	if err != nil {
		return err
	}
	return doDownloadEntities(fs, dtClient, options)
}

func (d DefaultCommand) DownloadEntities(fs afero.Fs, cmdOptions entitiesDirectDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	errors := validateParameters(cmdOptions.environmentUrl, cmdOptions.projectName)

	if len(errors) > 0 {
		return PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadEntitiesOptions{
		downloadOptionsShared: downloadOptionsShared{
			environmentUrl: cmdOptions.environmentUrl,
			auth: manifest.Auth{
				Token: manifest.AuthSecret{
					Name:  cmdOptions.envVarName,
					Value: token,
				},
			},
			outputFolder:            cmdOptions.outputFolder,
			projectName:             cmdOptions.projectName,
			forceOverwriteManifest:  cmdOptions.forceOverwrite,
			concurrentDownloadLimit: concurrentDownloadLimit,
		},
		specificEntitiesTypes: cmdOptions.specificEntitiesTypes,
	}

	dtClient, err := client.NewClassicClient(cmdOptions.environmentUrl, token)
	if err != nil {
		return err
	}

	return doDownloadEntities(fs, dtClient, options)
}

func doDownloadEntities(fs afero.Fs, dtClient client.Client, opts downloadEntitiesOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)

	downloadedConfigs := downloadEntities(dtClient, opts)

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

func downloadEntities(dtClient client.Client, opts downloadEntitiesOptions) project.ConfigsPerType {
	dtClient = client.LimitClientParallelRequests(dtClient, opts.downloadOptionsShared.concurrentDownloadLimit)

	var entitiesObjects project.ConfigsPerType

	// download specific entity types only
	if len(opts.specificEntitiesTypes) > 0 {
		log.Debug("Entity Types to download: \n - %v", strings.Join(opts.specificEntitiesTypes, "\n - "))
		entitiesObjects = entities.Download(dtClient, opts.specificEntitiesTypes, opts.projectName)
	} else {
		entitiesObjects = entities.DownloadAll(dtClient, opts.downloadOptionsShared.projectName)
	}

	if numEntities := sumConfigs(entitiesObjects); numEntities > 0 {
		log.Info("Downloaded %d entities types.", numEntities)
	} else {
		log.Info("No entities were found. No files will be created.")
		return nil
	}

	return entitiesObjects
}
