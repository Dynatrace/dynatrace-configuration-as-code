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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"os"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/entities"
	project "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"github.com/spf13/afero"
)

type entitiesDownloadCommandOptions struct {
	downloadCommandOptionsShared
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

func (d DefaultCommand) DownloadEntitiesBasedOnManifest(fs afero.Fs, cmdOptions entitiesManifestDownloadOptions) error {

	envUrl, token, tokenEnvVar, err := getEnvFromManifest(fs, cmdOptions.manifestFile, cmdOptions.specificEnvironmentName, cmdOptions.projectName)
	if err != nil {
		return err
	}

	if !cmdOptions.forceOverwrite {
		cmdOptions.projectName = fmt.Sprintf("%s_%s", cmdOptions.projectName, cmdOptions.specificEnvironmentName)
	}

	concurrentDownloadLimit := concurrentRequestLimitFromEnv()

	options := downloadOptionsShared{
		environmentUrl:          envUrl,
		token:                   token,
		tokenEnvVarName:         tokenEnvVar,
		outputFolder:            cmdOptions.outputFolder,
		projectName:             cmdOptions.projectName,
		forceOverwriteManifest:  cmdOptions.forceOverwrite,
		clientProvider:          client.NewDynatraceClient,
		concurrentDownloadLimit: concurrentDownloadLimit,
	}
	return doDownloadEntities(fs, options)
}

func (d DefaultCommand) DownloadEntities(fs afero.Fs, cmdOptions entitiesDirectDownloadOptions) error {
	token := os.Getenv(cmdOptions.envVarName)
	concurrentDownloadLimit := concurrentRequestLimitFromEnv()
	errors := validateParameters(cmdOptions.envVarName, cmdOptions.environmentUrl, cmdOptions.projectName, token)

	if len(errors) > 0 {
		return PrintAndFormatErrors(errors, "not all necessary information is present to start downloading configurations")
	}

	options := downloadOptionsShared{
		environmentUrl:          cmdOptions.environmentUrl,
		token:                   token,
		tokenEnvVarName:         cmdOptions.envVarName,
		outputFolder:            cmdOptions.outputFolder,
		projectName:             cmdOptions.projectName,
		forceOverwriteManifest:  cmdOptions.forceOverwrite,
		clientProvider:          client.NewDynatraceClient,
		concurrentDownloadLimit: concurrentDownloadLimit,
	}
	return doDownloadEntities(fs, options)
}

func doDownloadEntities(fs afero.Fs, opts downloadOptionsShared) error {
	err := preDownloadValidations(fs, opts)
	if err != nil {
		return err
	}

	log.Info("Downloading from environment '%v' into project '%v'", opts.environmentUrl, opts.projectName)

	downloadedConfigs, err := downloadEntities(opts)

	if err != nil {
		return err
	}

	return writeConfigs(downloadedConfigs, opts, err, fs)
}

func downloadEntities(opts downloadOptionsShared) (project.ConfigsPerType, error) {

	c, err := opts.getDynatraceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Dynatrace client: %w", err)
	}

	c = client.LimitClientParallelRequests(c, opts.concurrentDownloadLimit)

	entitiesObjects := entities.DownloadAll(c, opts.projectName)

	if numEntities := sumConfigs(entitiesObjects); numEntities > 0 {
		log.Info("Downloaded %d entities types.", numEntities)
	} else {
		log.Info("No entities were found. No files will be created.")
		return nil, nil
	}

	return entitiesObjects, nil
}
