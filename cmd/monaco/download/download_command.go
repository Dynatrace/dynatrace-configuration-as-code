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
	utilEnv "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
)

func GetDownloadCommand(fs afero.Fs, command Command) (downloadCmd *cobra.Command) {

	downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

Either downloading based on an existing manifest, or by defining environment URL and API token via the 'direct' sub-command.

To download entities, use download entities`,
		Example: `- monaco download manifest manifest.yaml some_environment_from_manifest
- monaco download direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'direct' or 'manifest' sub-command is required")
		},
	}

	GetDownloadConfigsCommand(fs, command, downloadCmd)

	if utilEnv.FeatureFlagEnabled("MONACO_FEAT_ENTITIES") {
		GetDownloadEntitiesCommand(fs, command, downloadCmd)
	}

	return downloadCmd
}

func GetDownloadConfigsCommand(fs afero.Fs, command Command, downloadCmd *cobra.Command) {
	var project, outputFolder string
	var forceOverwrite bool
	var specificApis []string
	var specificSettings []string
	var onlyAPIs bool

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
					downloadCommandOptionsShared: downloadCommandOptionsShared{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
					specificAPIs:    specificApis,
					specificSchemas: specificSettings,
					onlyAPIs:        onlyAPIs,
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
					downloadCommandOptionsShared: downloadCommandOptionsShared{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
					specificAPIs:    specificApis,
					specificSchemas: specificSettings,
					onlyAPIs:        onlyAPIs,
				},
			}
			return command.DownloadConfigs(fs, options)

		},
	}

	setupSharedConfigsFlags(manifestDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificApis, &specificSettings, &onlyAPIs)
	setupSharedConfigsFlags(directDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificApis, &specificSettings, &onlyAPIs)

	downloadCmd.AddCommand(manifestDownloadCmd)
	downloadCmd.AddCommand(directDownloadCmd)
}

func GetDownloadEntitiesCommand(fs afero.Fs, command Command, downloadCmd *cobra.Command) {
	var project, outputFolder string
	var forceOverwrite bool

	downloadEntitiesCmd := &cobra.Command{
		Use:   "entities",
		Short: "Download entities configuration from Dynatrace",
		Long: `Download entities configuration from Dynatrace

Either downloading based on an existing manifest, or by defining environment URL and API token via the 'direct' sub-command.`,
		Example: `- monaco download entities manifest manifest.yaml some_environment_from_manifest
- monaco download entities direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("'direct' or 'manifest' sub-command is required")
		},
	}

	manifestDownloadCmd := &cobra.Command{
		Use:     "manifest [manifest file] [environment to download]",
		Aliases: []string{"m"},
		Short:   "Download configuration from Dynatrace via a manifest file",
		Example: `monaco download entities manifest.yaml some_environment_from_manifest`,
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
			options := entitiesManifestDownloadOptions{
				manifestFile:            manifest,
				specificEnvironmentName: specificEnvironment,
				entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
					downloadCommandOptionsShared: downloadCommandOptionsShared{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
				},
			}
			return command.DownloadEntitiesBasedOnManifest(fs, options)
		},
	}

	directDownloadCmd := &cobra.Command{
		Use:     "direct [URL] [TOKEN_NAME]",
		Aliases: []string{"d"},
		Short:   "Download configuration from a Dynatrace environment specified on the command line",
		Example: `monaco download entities direct https://environment.live.dynatrace.com API_TOKEN_ENV_VAR_NAME`,
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
			options := entitiesDirectDownloadOptions{
				environmentUrl: url,
				envVarName:     tokenEnvVar,
				entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
					downloadCommandOptionsShared: downloadCommandOptionsShared{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
				},
			}
			return command.DownloadEntities(fs, options)

		},
	}

	setupSharedEntitiesFlags(manifestDownloadCmd, &project, &outputFolder, &forceOverwrite)
	setupSharedEntitiesFlags(directDownloadCmd, &project, &outputFolder, &forceOverwrite)

	downloadEntitiesCmd.AddCommand(manifestDownloadCmd)
	downloadEntitiesCmd.AddCommand(directDownloadCmd)

	downloadCmd.AddCommand(downloadEntitiesCmd)
}

func setupSharedConfigsFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool, specificApis *[]string, specificSettings *[]string, onlyAPIs *bool) {
	setupSharedFlags(cmd, project, outputFolder, forceOverwrite)
	// flags always available
	cmd.Flags().StringSliceVarP(specificApis, "specific-apis", "a", make([]string, 0), "List of APIs to download")
	cmd.Flags().StringSliceVarP(specificSettings, "specific-settings", "s", make([]string, 0), "List of settings 2.0 schema IDs specifying which Settings 2.0 objects to download")
	cmd.Flags().BoolVar(onlyAPIs, "only-apis", false, "Only download config APIs, skip downloading settings 2.0 objects")
	cmd.MarkFlagsMutuallyExclusive("specific-settings", "only-apis")

	err := cmd.RegisterFlagCompletionFunc("specific-apis", completion.AllAvailableApis)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
}

func setupSharedEntitiesFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool) {
	setupSharedFlags(cmd, project, outputFolder, forceOverwrite)
}

func setupSharedFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool) {
	// flags always available
	cmd.Flags().StringVarP(project, "project", "p", "project", "Project to create within the output-folder")
	cmd.Flags().StringVarP(outputFolder, "output-folder", "o", "", "Folder to write downloaded configs to")
	cmd.Flags().BoolVarP(forceOverwrite, "force", "f", false, "Force overwrite any existing manifest.yaml, rather than creating an additional manifest_{timestamp}.yaml. Manifest download: additionally never append source environment name to project folder name")
	err := cmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
}
