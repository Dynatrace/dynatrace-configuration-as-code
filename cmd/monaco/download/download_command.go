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
	TokenFlag                     = "token"
	OAuthIdFlag                   = "oauth-client-id"
	OAuthSecretFlag               = "oauth-client-secret"
	PlatformTokenFlag             = "platform-token"
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

	cmd = &cobra.Command{
		Short: "Download configuration from Dynatrace",
		Long: `Download configuration from Dynatrace

  Either downloading based on an existing manifest, or define an URL pointing to an environment to download configuration from.`,

		Use: "download",
		Example: fmt.Sprintf(`  # download from  specific environment defined in manifest.yaml
  monaco download [--%s manifest.yaml] --%s MY_ENV ...

  # download without manifest
  monaco download --%s url --%s DT_TOKEN [--%s CLIENT_ID --%s CLIENT_SECRET] ...`, ManifestFlag, EnvironmentFlag, UrlFlag, TokenFlag, OAuthIdFlag, OAuthSecretFlag),

		PreRunE: func(cmd *cobra.Command, args []string) error {
			return preRunChecks(f)
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
			}

			if f.environmentURL != "" {
				f.manifestFile = ""
				return command.DownloadConfigs(cmd.Context(), fs, f)
			}
			return command.DownloadConfigsBasedOnManifest(cmd.Context(), fs, f)
		},
	}

	setupSharedFlags(cmd, &f.projectName, &f.outputFolder, &f.forceOverwrite)

	// download via manifest
	cmd.Flags().StringVarP(&f.manifestFile, ManifestFlag, "m", "manifest.yaml", "Name (and the path) to the manifest file. Defaults to 'manifest.yaml'.")
	cmd.Flags().StringVarP(&f.specificEnvironmentName, EnvironmentFlag, "e", "", "Specify an environment defined in the manifest to download the configurations.")
	// download without manifest
	cmd.Flags().StringVar(&f.environmentURL, UrlFlag, "", "URL to the Dynatrace environment from which to download the configuration. "+
		fmt.Sprintf("To be able to connect to any Dynatrace environment, an API-Token needs to be provided using '--%s'. ", TokenFlag)+
		fmt.Sprintf("In case of connecting to a Dynatrace Platform, an OAuth Client ID, as well as an OAuth Client Secret, needs to be provided as well using the flags '--%s' and '--%s'. ", OAuthIdFlag, OAuthSecretFlag)+
		fmt.Sprintf("This flag is not combinable with the flag '--%s.'", ManifestFlag))
	cmd.Flags().StringVar(&f.token, TokenFlag, "", fmt.Sprintf("API-Token environment variable. Required when using the flag '--%s'", UrlFlag))
	cmd.Flags().StringVar(&f.clientID, OAuthIdFlag, "", fmt.Sprintf("OAuth client ID environment variable. This flag and '--%s' or '--%s' is required when using the flag '--%s' and connecting to a Dynatrace Platform.", OAuthSecretFlag, PlatformTokenFlag, UrlFlag))
	cmd.Flags().StringVar(&f.clientSecret, OAuthSecretFlag, "", fmt.Sprintf("OAuth client secret environment variable. This flag and '--%s' or '--%s' is required when using the flag '--%s' and connecting to a Dynatrace Platform.", OAuthIdFlag, PlatformTokenFlag, UrlFlag))
	cmd.Flags().StringVar(&f.platformToken, PlatformTokenFlag, "", fmt.Sprintf("Platform token environment variable. This flag or '--%s' is required when using  and connecting to a Dynatrace Platform.", UrlFlag))

	// download options
	cmd.Flags().StringSliceVarP(&f.specificAPIs, ApiFlag, "a", nil, "Download one or more classic configuration APIs, including deprecated ones. (Repeat flag or use comma-separated values)")
	cmd.Flags().StringSliceVarP(&f.specificSchemas, SettingsSchemaFlag, "s", nil, "Download settings 2.0 objects of one or more settings 2.0 schemas. (Repeat flag or use comma-separated values)")
	cmd.Flags().BoolVar(&onlyApis, OnlyApisFlag, false, "Download only classic configuration APIs. Deprecated configuration APIs will not be included.")
	cmd.Flags().BoolVar(&onlySettings, OnlySettingsFlag, false, "Download only settings 2.0 objects")
	cmd.Flags().BoolVar(&onlyAutomation, OnlyAutomationFlag, false, "Only download automation objects, skip all other configuration types")
	cmd.Flags().BoolVar(&onlyDocuments, OnlyDocumentsFlag, false, "Only download documents, skip all other configuration types")
	cmd.Flags().BoolVar(&onlyBuckets, OnlyBucketsFlag, false, "Only download buckets, skip all other configuration types")

	// combinations
	cmd.MarkFlagsMutuallyExclusive(SettingsSchemaFlag, OnlySettingsFlag)
	cmd.MarkFlagsMutuallyExclusive(ApiFlag, OnlyApisFlag)

	if featureflags.OpenPipeline.Enabled() {
		cmd.Flags().BoolVar(&onlyOpenPipeline, OnlyOpenPipelineFlag, false, "Only download openpipeline configurations, skip all other configuration types")
	}

	if featureflags.Segments.Enabled() {
		cmd.Flags().BoolVar(&onlySegments, OnlySegmentsFlag, false, "Only download segment configurations, skip all other configuration types")
	}

	if featureflags.ServiceLevelObjective.Enabled() {
		cmd.Flags().BoolVar(&onlySloV2, OnlySloV2Flag, false, fmt.Sprintf("Only download %s, skip all other configuration types", config.ServiceLevelObjectiveID))
	}

	err := errors.Join(
		cmd.RegisterFlagCompletionFunc(TokenFlag, completion.EnvVarName),
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

func preRunChecks(f downloadCmdOptions) error {
	switch {
	case f.environmentURL != "" && f.manifestFile != "manifest.yaml":
		return fmt.Errorf("'%s' and '%s' are mutually exclusive", UrlFlag, ManifestFlag)
	case f.environmentURL != "" && f.specificEnvironmentName != "":
		return fmt.Errorf("'%s' is specific to manifest-based download and incompatible with direct download from '%s'", EnvironmentFlag, UrlFlag)
	case f.environmentURL != "":
		switch {
		case f.token == "":
			return fmt.Errorf("if '%s' is set, '%s' also must be set", UrlFlag, TokenFlag)
		case (f.clientID == "") != (f.clientSecret == ""):
			return fmt.Errorf("'%s' and '%s' must always be set together", OAuthIdFlag, OAuthSecretFlag)
		default:
			return nil
		}
	case f.manifestFile != "":
		switch {
		case f.token != "" || f.clientID != "" || f.clientSecret != "":
			return fmt.Errorf("'%s', '%s' and '%s' can only be used with '%s', while '%s' must NOT be set ", TokenFlag, OAuthIdFlag, OAuthSecretFlag, UrlFlag, ManifestFlag)
		case f.specificEnvironmentName == "":
			return fmt.Errorf("to download with manifest, '%s' needs to be specified", EnvironmentFlag)
		}
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
