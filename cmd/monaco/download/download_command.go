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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner/completion"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

func GetDownloadCommand(fs afero.Fs, command Command) (downloadCmd *cobra.Command) {
	var specificEnvironment, project, outputFolder string
	var specificApis []string

	downloadCmd = &cobra.Command{
		Use:   "download [manifest]",
		Short: "Download configuration from Dynatrace via a manifest file",
		Long: `Download configuration from Dynatrace

Either downloading based on an existing manifest, or by defining environment URL and API token via the 'direct' sub-command.`,
		Example: `- monaco download manifest.yaml -s some_environment_from_manifest
- monaco download direct environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 || args[0] == "" {
				return fmt.Errorf(`manifest has to be provided as argument`)
			}
			return nil
		},
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			return files.YamlExtensions, cobra.ShellCompDirectiveDefault
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest := args[0]
			return command.DownloadConfigsBasedOnManifest(fs, manifest, project, specificEnvironment, outputFolder, specificApis)
		},
	}

	directDownloadCmd := &cobra.Command{
		Use:     "direct [URL] [TOKEN_NAME]",
		Short:   "Download configuration from a Dynatrace environment specified on the command line",
		Long:    `Download configuration from a Dynatrace environment specified on the command line`,
		Example: `monaco download direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return fmt.Errorf(`url and token have to be provided as positional argument`)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			tokenEnvVar := args[1]
			return command.DownloadConfigs(fs, url, project, tokenEnvVar, outputFolder, specificApis)

		},
	}

	downloadCmd.AddCommand(directDownloadCmd)

	// download using the manifest
	downloadCmd.Flags().StringVarP(&specificEnvironment, "specific-environment", "s", "", "Specific environment from Manifest to download")
	err := downloadCmd.MarkFlagRequired("specific-environment")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	err = downloadCmd.RegisterFlagCompletionFunc("specific-environment", completion.EnvironmentByArg0)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	setupSharedFlags(downloadCmd, &project, &outputFolder, &specificApis)
	setupSharedFlags(directDownloadCmd, &project, &outputFolder, &specificApis)

	return downloadCmd
}

func setupSharedFlags(cmd *cobra.Command, project, outputFolder *string, specificApis *[]string) {
	// flags always available
	cmd.Flags().StringSliceVarP(specificApis, "specific-api", "a", make([]string, 0), "APIs to download")
	cmd.Flags().StringVarP(project, "project", "p", "project", "Project to create within the output-folder")
	cmd.Flags().StringVarP(outputFolder, "output-folder", "o", "", "Folder to write downloaded configs to")
	err := cmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	err = cmd.RegisterFlagCompletionFunc("specific-api", completion.AllAvailableApis)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
}
