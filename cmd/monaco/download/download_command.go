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
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"net/http"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner/completion"
)

func GetDownloadCommand(fs afero.Fs, command Command) (cmd *cobra.Command) {
	var f downloadCmdOptions

	cmd = &cobra.Command{
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

  Either downloading based on an existing manifest, or define an URL pointing to an environment to download configuration from.`,

		Use: "download",
		Example: `  # download from  specific environment defined in manifest.yaml
  monaco download [--manifest manifest.yaml] --environment MY_ENV ...

  # download without manifest
  monaco download --url url --token DT_TOKEN [--oauth-client-id CLIENT_ID --oauth-client-secret CLIENT_SECRET] ...`,

		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunChecks(f)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if f.environmentURL != "" {
				f.manifestFile = ""
				return command.DownloadConfigs(fs, f)
			}
			return command.DownloadConfigsBasedOnManifest(fs, f)
		},
	}

	setupSharedFlags(cmd, &f.projectName, &f.outputFolder, &f.forceOverwrite)

	// download via manifest
	cmd.Flags().StringVarP(&f.manifestFile, "manifest", "m", "manifest.yaml", "Name (and the path) to the manifest file. If not provided \"manifest.yaml\" value will be used.")
	cmd.Flags().StringVarP(&f.specificEnvironmentName, "environment", "e", "", "Specify a concrete environment defined in the manifest file that shall be downloaded")
	// download without manifest
	cmd.Flags().StringVar(&f.environmentURL, "url", "", "URL to the dynatrace environment from which to download configuration from. To be able to connect, token and, in case of connecting to platform, a pari of OAuth client ID na client secret needs to bi provide via adequate flags (\"--token\", \"--oauth-client-id\", \"--oauth-client-secret\"). Not able to combine with \"--manifest\".")
	cmd.Flags().StringVar(&f.token, "token", "", "Token secret to connect to DT server. Use only with \"--url\"")
	cmd.Flags().StringVar(&f.clientID, "oauth-client-id", "", "OAuth client ID is used to connect to DT server via OAuth. Use only with \"--url\"")
	cmd.Flags().StringVar(&f.clientSecret, "oauth-client-secret", "", "OAuth client secret is used to connect to DT server via OAuth. Use only with \"--url\"")

	// download options
	cmd.Flags().StringSliceVarP(&f.specificAPIs, "api", "a", nil, "One or more APIs to download (flag can be repeated or value defined as comma-separated list)")
	cmd.Flags().StringSliceVarP(&f.specificSchemas, "settings-schema", "s", nil, "One or more settings 2.0 schemas to download (flag can be repeated or value defined as comma-separated list)")

	cmd.Flags().BoolVar(&f.onlyAPIs, "only-apis", false, "Only download config APIs, skip downloading settings 2.0 objects")
	cmd.Flags().BoolVar(&f.onlySettings, "only-settings", false, "Only download settings 2.0 objects, skip downloading config APIs")
	cmd.MarkFlagsMutuallyExclusive("settings-schema", "only-apis", "only-settings")
	cmd.MarkFlagsMutuallyExclusive("api", "only-apis", "only-settings")
	cmd.MarkFlagsMutuallyExclusive("only-apis", "only-settings")

	if featureflags.Entities().Enabled() {
		getDownloadEntitiesCommand(fs, command, cmd)
	}

	err := cmd.RegisterFlagCompletionFunc("api", completion.AllAvailableApis)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return cmd
}

func preRunChecks(f downloadCmdOptions) error {
	switch {
	case f.environmentURL != "" && f.manifestFile != "manifest.yaml":
		return errors.New("\"url\" and \"token\" are mutually exclusive")
	case f.environmentURL != "" && f.specificEnvironmentName != "":
		return errors.New("")
	case f.environmentURL != "":
		switch {
		case f.token == "":
			return errors.New("if \"url\" is set, \"token\" also must be set")
		case (f.clientID == "") != (f.clientSecret == ""):
			return errors.New("\"oauth-client-id\" and \"oauth-client-secret\" always come in pair")
		default:
			return nil
		}
	case f.manifestFile != "":
		switch {
		case f.token != "" || f.clientID != "" || f.clientSecret != "":
			return errors.New("\"token\", \"oauth-client-id\" and \"oauth-client-secret\" can only be used with \"url\", while \"manifest\" must NOT be set ")
		case f.specificEnvironmentName == "":
			return errors.New("to download with manifest, \"environment\" needs to be specified")
		}
	}

	return nil
}

func getDownloadEntitiesCommand(fs afero.Fs, command Command, downloadCmd *cobra.Command) {
	var project, outputFolder string
	var forceOverwrite bool
	var specificEntitiesTypes []string

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
			m := args[0]
			specificEnvironment := args[1]
			options := entitiesManifestDownloadOptions{
				manifestFile:            m,
				specificEnvironmentName: specificEnvironment,
				entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
					sharedDownloadCmdOptions: sharedDownloadCmdOptions{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
					specificEntitiesTypes: specificEntitiesTypes,
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
				environmentURL: url,
				envVarName:     tokenEnvVar,
				entitiesDownloadCommandOptions: entitiesDownloadCommandOptions{
					sharedDownloadCmdOptions: sharedDownloadCmdOptions{
						projectName:    project,
						outputFolder:   outputFolder,
						forceOverwrite: forceOverwrite,
					},
					specificEntitiesTypes: specificEntitiesTypes,
				},
			}
			return command.DownloadEntities(fs, options)

		},
	}

	setupSharedEntitiesFlags(manifestDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificEntitiesTypes)
	setupSharedEntitiesFlags(directDownloadCmd, &project, &outputFolder, &forceOverwrite, &specificEntitiesTypes)

	downloadEntitiesCmd.AddCommand(manifestDownloadCmd)
	downloadEntitiesCmd.AddCommand(directDownloadCmd)

	downloadCmd.AddCommand(downloadEntitiesCmd)
}

func setupSharedEntitiesFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool, specificEntitiesTypes *[]string) {
	setupSharedFlags(cmd, project, outputFolder, forceOverwrite)
	cmd.Flags().StringSliceVarP(specificEntitiesTypes, "specific-types", "s", make([]string, 0), "List of entity type IDs specifying which entity types to download")

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

// printUploadToSameEnvironmentWarning function may display a warning message on the console,
// notifying the user that downloaded objects cannot be uploaded to the same environment.
// It verifies the version of the tenant and, depending on the result, it may or may not display the warning.
func printUploadToSameEnvironmentWarning(env manifest.EnvironmentDefinition) {
	var serverVersion version.Version
	var err error

	var httpClient *http.Client
	if env.Auth.OAuth == nil {
		httpClient = client.NewTokenAuthClient(env.Auth.Token.Value)
	} else {
		credentials := client.OauthCredentials{
			ClientID:     env.Auth.OAuth.ClientID.Value,
			ClientSecret: env.Auth.OAuth.ClientSecret.Value,
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}
		httpClient = client.NewOAuthClient(context.TODO(), credentials)
	}

	serverVersion, err = client.GetDynatraceVersion(httpClient, env.URL.Value)
	if err != nil {
		log.Warn("Unable to determine server version %q: %w", env.URL.Value, err)
		return
	}
	if serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262}) {
		logUploadToSameEnvironmentWarning()
	}
}

func logUploadToSameEnvironmentWarning() {
	log.Warn("Uploading Settings 2.0 objects to the same environment is not possible due to your cluster version " +
		"being below 1.262.0, which Monaco does not support for reliably updating downloaded settings without having " +
		"duplicate configurations. Consider upgrading to 1.262+")
}
