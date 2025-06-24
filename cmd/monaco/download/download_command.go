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
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

type OnlyFlag = string

const (
	EnvironmentFlag               = "environment"
	UrlFlag                       = "url"
	ManifestFlag                  = "manifest"
	AccessTokenFlag               = "token"
	PlatformTokenFlag             = "platform-token"
	OAuthIdFlag                   = "oauth-client-id"
	OAuthSecretFlag               = "oauth-client-secret"
	ApiFlag                       = "api"
	SettingsSchemaFlag            = "settings-schema"
	ProjectFlag                   = "project"
	OutputFolderFlag              = "output-folder"
	ForceFlag                     = "force"
	OnlyApisFlag         OnlyFlag = "only-apis"
	OnlySettingsFlag     OnlyFlag = "only-settings"
	OnlyAutomationFlag   OnlyFlag = "only-automation"
	OnlyDocumentsFlag    OnlyFlag = "only-documents"
	OnlyBucketsFlag      OnlyFlag = "only-buckets"
	OnlyOpenPipelineFlag OnlyFlag = "only-openpipeline"
	OnlySloV2Flag        OnlyFlag = "only-slo-v2"
	OnlySegmentsFlag     OnlyFlag = "only-segments"
)

func GetDownloadCommand(fs afero.Fs, command Command) (cmd *cobra.Command) {
	var f downloadCmdOptions
	var onlySettings, onlyApis, onlyOpenPipeline, onlySegments, onlySloV2, onlyDocuments, onlyBuckets, onlyAutomation bool

	platformTokenAddendum := ""
	if featureflags.PlatformToken.Enabled() {
		platformTokenAddendum = fmt.Sprintf(" [--%s PLATFORM_TOKEN]", PlatformTokenFlag)
	}

	cmd = &cobra.Command{
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

  Either downloading based on an existing manifest, or define an URL pointing to an environment to download configuration from.`,

		Use: "download",
		Example: fmt.Sprintf(`  # download from  specific environment defined in manifest.yaml
  monaco download [--%s manifest.yaml] --%s MY_ENV ...

  # download without manifest
  monaco download --%s url [--%s DT_TOKEN] [--%s CLIENT_ID --%s CLIENT_SECRET]%s ...`, ManifestFlag, EnvironmentFlag, UrlFlag, AccessTokenFlag, OAuthIdFlag, OAuthSecretFlag, platformTokenAddendum),

		PreRunE: func(cmd *cobra.Command, args []string) error {
			if f.environmentURL != "" {
				return preRunChecksForDirectDownload(f)
			}

			return preRunChecksForManifestDownload(f)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			f.onlyOptions = OnlyOptions{
				OnlySettingsFlag:     onlySettings || len(f.specificSchemas) > 0,
				OnlyApisFlag:         onlyApis || len(f.specificAPIs) > 0,
				OnlySegmentsFlag:     onlySegments,
				OnlySloV2Flag:        onlySloV2,
				OnlyOpenPipelineFlag: onlyOpenPipeline,
				OnlyDocumentsFlag:    onlyDocuments,
				OnlyBucketsFlag:      onlyBuckets,
				OnlyAutomationFlag:   onlyAutomation,
			}

			if f.environmentURL != "" {
				return command.DownloadConfigs(cmd.Context(), fs, f)
			}

			if f.manifestFile == "" {
				f.manifestFile = "manifest.yaml"
			}
			return command.DownloadConfigsBasedOnManifest(cmd.Context(), fs, f)
		},
	}

	setupSharedFlags(cmd, &f.projectName, &f.outputFolder, &f.forceOverwrite)

	// download via manifest
	cmd.Flags().StringVarP(&f.manifestFile, ManifestFlag, "m", "", "Name (and the path) to the manifest file. If not specified, 'manifest.yaml' will be used.")
	cmd.Flags().StringVarP(&f.specificEnvironmentName, EnvironmentFlag, "e", "", "Specify an environment defined in the manifest to download the configurations.")
	// download without manifest
	cmd.Flags().StringVar(&f.environmentURL, UrlFlag, "", "URL to the Dynatrace environment from which to download the configuration. "+
		fmt.Sprintf("To be able to connect to any Dynatrace environment, an access token needs to be provided using '--%s'. ", AccessTokenFlag)+
		fmt.Sprintf("In case of connecting to a Dynatrace Platform, an OAuth Client ID, as well as an OAuth Client Secret, needs to be provided as well using the flags '--%s' and '--%s'. ", OAuthIdFlag, OAuthSecretFlag)+
		fmt.Sprintf("This flag is not combinable with the flag '--%s.'", ManifestFlag))
	cmd.Flags().StringVar(&f.accessToken, AccessTokenFlag, "", fmt.Sprintf("Access token environment variable. Required when using the flag '--%s' and downloading Dynatrace Classic configurations.", UrlFlag))
	cmd.Flags().StringVar(&f.clientID, OAuthIdFlag, "", fmt.Sprintf("OAuth client ID environment variable. For use with '--%s' when using the flag '--%s' and to download Dynatrace Platform configurations.", OAuthSecretFlag, UrlFlag))
	cmd.Flags().StringVar(&f.clientSecret, OAuthSecretFlag, "", fmt.Sprintf("OAuth client secret environment variable. For use with '--%s' when using the flag '--%s' and to download Dynatrace Platform configurations.", OAuthIdFlag, UrlFlag))
	if featureflags.PlatformToken.Enabled() {
		cmd.Flags().StringVar(&f.platformToken, PlatformTokenFlag, "", fmt.Sprintf("Platform token environment variable. For use when using the flag '--%s' and to download Dynatrace Platform configurations.", UrlFlag))
	}

	// download options
	cmd.Flags().StringSliceVarP(&f.specificAPIs, ApiFlag, "a", nil, "Download one or more classic configuration APIs, including deprecated ones. (Repeat flag or use comma-separated values)")
	cmd.Flags().StringSliceVarP(&f.specificSchemas, SettingsSchemaFlag, "s", nil, "Download settings 2.0 objects of one or more settings 2.0 schemas. (Repeat flag or use comma-separated values)")
	cmd.Flags().BoolVar(&onlyApis, OnlyApisFlag, false, "Download only classic configuration APIs. Deprecated configuration APIs will not be included.")
	cmd.Flags().BoolVar(&onlySettings, OnlySettingsFlag, false, "Download only settings 2.0 objects")
	cmd.Flags().BoolVar(&onlyAutomation, OnlyAutomationFlag, false, "Only download automation objects")
	cmd.Flags().BoolVar(&onlyDocuments, OnlyDocumentsFlag, false, "Only download documents")
	cmd.Flags().BoolVar(&onlyBuckets, OnlyBucketsFlag, false, "Only download buckets")
	cmd.Flags().BoolVar(&onlyOpenPipeline, OnlyOpenPipelineFlag, false, "Only download openpipeline configurations")

	// combinations
	cmd.MarkFlagsMutuallyExclusive(SettingsSchemaFlag, OnlySettingsFlag)
	cmd.MarkFlagsMutuallyExclusive(ApiFlag, OnlyApisFlag)

	if featureflags.Segments.Enabled() {
		cmd.Flags().BoolVar(&onlySegments, OnlySegmentsFlag, false, "Only download segment configurations")
	}

	if featureflags.ServiceLevelObjective.Enabled() {
		cmd.Flags().BoolVar(&onlySloV2, OnlySloV2Flag, false, fmt.Sprintf("Only download %s configurations", config.ServiceLevelObjectiveID))
	}

	err := errors.Join(
		cmd.RegisterFlagCompletionFunc(AccessTokenFlag, completion.EnvVarName),
		cmd.RegisterFlagCompletionFunc(OAuthIdFlag, completion.EnvVarName),
		cmd.RegisterFlagCompletionFunc(OAuthSecretFlag, completion.EnvVarName),

		cmd.RegisterFlagCompletionFunc(ManifestFlag, completion.YamlFile),

		cmd.RegisterFlagCompletionFunc(ApiFlag, completion.AllAvailableApis),
	)

	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return cmd
}

func preRunChecksForDirectDownload(f downloadCmdOptions) error {
	if f.manifestFile != "" {
		return fmt.Errorf("'--%s' and '--%s' are mutually exclusive", UrlFlag, ManifestFlag)
	}

	if f.specificEnvironmentName != "" {
		return fmt.Errorf("'--%s' is specific to manifest-based download and incompatible with direct download with '--%s'", EnvironmentFlag, UrlFlag)
	}

	if (f.accessToken == "") && (f.clientID == "") && (f.clientSecret == "") && (f.platformToken == "") {
		if featureflags.PlatformToken.Enabled() {
			return fmt.Errorf("if '--%s' is set, '--%s', or '--%s' and '--%s', or '--%s' must also be set", UrlFlag, AccessTokenFlag, OAuthIdFlag, OAuthSecretFlag, PlatformTokenFlag)
		}
		return fmt.Errorf("if '--%s' is set, '--%s' or '--%s' and '--%s' must also be set", UrlFlag, AccessTokenFlag, OAuthIdFlag, OAuthSecretFlag)
	}

	if (f.clientID == "") != (f.clientSecret == "") {
		return fmt.Errorf("'--%s' and '--%s' must always be set together", OAuthIdFlag, OAuthSecretFlag)
	}

	if (f.clientID != "") && (f.clientSecret != "") && (f.platformToken != "") {
		return fmt.Errorf("OAuth credentials and a platform token can't be used together")
	}

	return nil
}

func preRunChecksForManifestDownload(f downloadCmdOptions) error {
	if f.accessToken != "" || f.clientID != "" || f.clientSecret != "" || f.platformToken != "" {
		if featureflags.PlatformToken.Enabled() {
			return fmt.Errorf("'--%s', '--%s', '--%s', and '--%s' can only be used with '--%s', while '--%s' must NOT be set", AccessTokenFlag, OAuthIdFlag, OAuthSecretFlag, PlatformTokenFlag, UrlFlag, ManifestFlag)
		}
		return fmt.Errorf("'--%s', '--%s', and '--%s' can only be used with '--%s', while '--%s' must NOT be set", AccessTokenFlag, OAuthIdFlag, OAuthSecretFlag, UrlFlag, ManifestFlag)
	}

	if f.specificEnvironmentName == "" {
		return fmt.Errorf("to download with manifest, '--%s' needs to be specified", EnvironmentFlag)
	}

	return nil
}

func setupSharedFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool) {
	// flags always available
	cmd.Flags().StringVarP(project, ProjectFlag, "p", "project", "Project to create within the output-folder")
	cmd.Flags().StringVarP(outputFolder, OutputFolderFlag, "o", "", "Folder to write downloaded configs to")
	cmd.Flags().BoolVarP(forceOverwrite, ForceFlag, "f", false, "Force overwrite any existing manifest.yaml, rather than creating an additional manifest_{timestamp}.yaml. Manifest download: Never append the source environment name to the project folder name.")

	err := cmd.MarkFlagDirname(OutputFolderFlag)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
}
