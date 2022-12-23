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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

func GetDownloadCommand(fs afero.Fs, command Command) (downloadCmd *cobra.Command) {
	var project, outputFolder string
	var forceOverwrite bool
	var specificApis []string
	var skipSettings bool

	downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

Either downloading based on an existing manifest, or by defining environment URL and API token via the 'direct' sub-command.`,
		Example: `- monaco download manifest manifest.yaml some_environment_from_manifest
- monaco download direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'direct' or 'manifest' sub-command is required")
		},
	}

	manifestDownloadCmd := &cobra.Command{
		Use:     "manifest [manifest file] [environment to download]",
		Aliases: []string{"m"},
		Short:   "Download configuration from Dynatrace via a manifest file",
		Example: `monaco download manifest.yaml some_environment_from_manifest`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return fmt.Errorf(`manifest and environment name have to be provided as positional arguments`)
			}
			return nil
		},
		ValidArgsFunction: completion.DownloadManifestCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest := args[0]
			specificEnvironment := args[1]
			options := manifestDownloadOptions{
				manifestFile:            manifest,
				specificEnvironmentName: specificEnvironment,
				downloadCommandOptions: downloadCommandOptions{
					projectName:        project,
					outputFolder:       outputFolder,
					forceOverwrite:     forceOverwrite,
					apiNamesToDownload: specificApis,
					skipSettings:       skipSettings,
				},
			}
			return command.DownloadConfigsBasedOnManifest(fs, options)
		},
	}

	directDownloadCmd := &cobra.Command{
		Use:     "direct [URL] [TOKEN_NAME]",
		Aliases: []string{"d"},
		Short:   "Download configuration from a Dynatrace environment specified on the command line",
		Example: `monaco download direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 || args[0] == "" || args[1] == "" {
				return fmt.Errorf(`url and token have to be provided as positional argument`)
			}
			return nil
		},
		ValidArgsFunction: completion.DownloadDirectCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			tokenEnvVar := args[1]
			options := directDownloadOptions{
				environmentUrl: url,
				envVarName:     tokenEnvVar,
				downloadCommandOptions: downloadCommandOptions{
					projectName:        project,
					outputFolder:       outputFolder,
					forceOverwrite:     forceOverwrite,
					apiNamesToDownload: specificApis,
					skipSettings:       skipSettings,
				},
			}
			return command.DownloadConfigs(fs, options)

		},
	}

	setupSharedFlags(manifestDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificApis, &skipSettings)
	setupSharedFlags(directDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificApis, &skipSettings)

	downloadCmd.AddCommand(manifestDownloadCmd)
	downloadCmd.AddCommand(directDownloadCmd)

	return downloadCmd
}

func setupSharedFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool, specificApis *[]string, skipSettings *bool) {
	// flags always available
	cmd.Flags().StringSliceVarP(specificApis, "specific-api", "a", make([]string, 0), "APIs to download")
	cmd.Flags().StringVarP(project, "project", "p", "project", "Project to create within the output-folder")
	cmd.Flags().StringVarP(outputFolder, "output-folder", "o", "", "Folder to write downloaded configs to")
	cmd.Flags().BoolVarP(forceOverwrite, "force", "f", false, "Force overwrite any existing manifest.yaml, rather than creating an additional manifest_{timestamp}.yaml. Manifest download: additionally never append source environment name to project folder name.")
	cmd.Flags().BoolVar(skipSettings, "skip-settings", false, "Skip downloading settings 2.0 objects ")
	err := cmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	err = cmd.RegisterFlagCompletionFunc("specific-api", completion.AllAvailableApis)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
}
