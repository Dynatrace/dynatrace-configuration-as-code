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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/entities"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

type entitiesDownloadCommandOptions struct {
	downloadCommandOptionsShared
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

	env, err := cmdutils.GetEnvFromManifest(fs, cmdOptions.manifestFile, cmdOptions.specificEnvironmentName)
	if err != nil {
		return err
	}

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)

	options := downloadEntitiesOptions{
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
		specificEntitiesTypes: cmdOptions.specificEntitiesTypes,
	}
	return doDownloadEntities(fs, options)
}

func (d DefaultCommand) DownloadEntities(fs afero.Fs, cmdOptions entitiesDirectDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := environment.GetEnvValueIntLog(environment.ConcurrentRequestsEnvKey)
	errors := validateParameters(cmdOptions.envVarName, cmdOptions.environmentUrl, cmdOptions.projectName, token)

	if len(errors) > 0 {
		return errutils.PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadEntitiesOptions{
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
		specificEntitiesTypes: cmdOptions.specificEntitiesTypes,
	}
	return doDownloadEntities(fs, options)
}

func doDownloadEntities(fs afero.Fs, opts downloadEntitiesOptions) error {
	err := preDownloadValidations(fs, opts.downloadOptionsShared)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)

	downloadedConfigs, err := downloadEntities(opts)

	if err != nil {
		return err
	}

	return writeConfigs(downloadedConfigs, opts.downloadOptionsShared, fs)
}

func downloadEntities(opts downloadEntitiesOptions) (project.ConfigsPerType, error) {

	c, err := opts.downloadOptionsShared.getDynatraceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	c = client.LimitClientParallelRequests(c, opts.downloadOptionsShared.concurrentDownloadLimit)

	var entitiesObjects project.ConfigsPerType

	// download specific entity types only
	if len(opts.specificEntitiesTypes) > 0 {
		log.Debug("Entity Types to download: \n - %v", strings.Join(opts.specificEntitiesTypes, "\n - "))
		entitiesObjects = entities.Download(c, opts.specificEntitiesTypes, opts.projectName)
	} else {
		entitiesObjects = entities.DownloadAll(c, opts.downloadOptionsShared.projectName)
	}

	if numEntities := sumConfigs(entitiesObjects); numEntities > 0 {
		log.Info("Downloaded %d entities types.", numEntities)
	} else {
		log.Info("No entities were found. No files will be created.")
		return nil, nil
	}

	return entitiesObjects, nil
}
