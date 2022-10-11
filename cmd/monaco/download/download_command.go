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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner/completion"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

func GetDownloadCommand(fs afero.Fs, command Command) (downloadCmd *cobra.Command) {
	var manifest, specificEnvironment, url, project, tokenEnvVar, outputFolder string
	var specificApis []string

	downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

Either downloading based on an existing manifest, or by defining environment URL and API token via flags.`,
		Example: `- monaco download -m manifest.yaml -s some_environment_from_manifest
- monaco download -u environment.live.dynatrace.com -t API_TOKEN_ENV_VAR_NAME`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if manifest != "" {
				return command.DownloadConfigsBasedOnManifest(fs, manifest, project, specificEnvironment, outputFolder, specificApis)
			}

			if url != "" {
				return command.DownloadConfigs(fs, url, project, tokenEnvVar, outputFolder, specificApis)
			}

			return fmt.Errorf(`either '--manifest' or '--url' has to be provided`)
		},
	}

	// flags always available
	downloadCmd.Flags().StringSliceVarP(&specificApis, "specific-api", "a", make([]string, 0), "APIs to download")
	downloadCmd.Flags().StringVarP(&project, "project", "p", "project", "Project to create within the output-folder")
	downloadCmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "", "Folder to write downloaded configs to")
	err := downloadCmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	// TODO david.laubreiter: Continue flag

	// download using the manifest
	downloadCmd.Flags().StringVarP(&manifest, "manifest", "m", "", "Manifest file")
	downloadCmd.Flags().StringVarP(&specificEnvironment, "specific-environment", "s", "", "Specific environment from Manifest to download")

	// download directly using flags
	downloadCmd.Flags().StringVarP(&url, "url", "u", "", "Environment Url")
	downloadCmd.Flags().StringVarP(&tokenEnvVar, "token", "t", "", "Name of the environment variable containing the token ")

	err = downloadCmd.RegisterFlagCompletionFunc("specific-environment", completion.EnvironmentByArg0)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	err = downloadCmd.MarkFlagFilename("manifest", files.YamlExtensions...)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	downloadCmd.MarkFlagsMutuallyExclusive("manifest", "url")
	downloadCmd.MarkFlagsMutuallyExclusive("manifest", "token")

	downloadCmd.MarkFlagsRequiredTogether("url", "token")
	downloadCmd.MarkFlagsRequiredTogether("manifest", "specific-environment") // make specific environment optional?

	err = downloadCmd.RegisterFlagCompletionFunc("specific-api", completion.AllAvailableApis)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return downloadCmd

}
