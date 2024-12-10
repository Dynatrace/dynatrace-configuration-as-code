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
	"net/http"
	"net/url"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	clientAuth "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/auth"
	versionClient "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
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
			cmd.SilenceUsage = true

			if f.environmentURL != "" {
				f.manifestFile = ""
				return command.DownloadConfigs(fs, f)
			}
			return command.DownloadConfigsBasedOnManifest(fs, f)
		},
	}

	setupSharedFlags(cmd, &f.projectName, &f.outputFolder, &f.forceOverwrite)

	// download via manifest
	cmd.Flags().StringVarP(&f.manifestFile, "manifest", "m", "manifest.yaml", "Name (and the path) to the manifest file. Defaults to 'manifest.yaml'.")
	cmd.Flags().StringVarP(&f.specificEnvironmentName, "environment", "e", "", "Specify an environment defined in the manifest to download the configurations.")
	// download without manifest
	cmd.Flags().StringVar(&f.environmentURL, "url", "", "URL to the Dynatrace environment from which to download the configuration. "+
		"To be able to connect to any Dynatrace environment, an API-Token needs to be provided using '--token'. "+
		"In case of connecting to a Dynatrace Platform, an OAuth Client ID, as well as an OAuth Client Secret, needs to be provided as well using the flags '--oauth-client-id' and '--oauth-client-secret'. "+
		"This flag is not combinable with the flag '--manifest.'")
	cmd.Flags().StringVar(&f.token, "token", "", "API-Token environment variable. Required when using the flag '--url'")
	cmd.Flags().StringVar(&f.clientID, "oauth-client-id", "", "OAuth client ID environment variable. Required when using the flag '--url' and connecting to a Dynatrace Platform.")
	cmd.Flags().StringVar(&f.clientSecret, "oauth-client-secret", "", "OAuth client secret environment variable. Required when using the flag '--url' and connecting to a Dynatrace Platform.")

	// download options
	cmd.Flags().StringSliceVarP(&f.specificAPIs, "api", "a", nil, "Download one or more classic configuration APIs, including deprecated ones. (Repeat flag or use comma-separated values)")
	cmd.Flags().StringSliceVarP(&f.specificSchemas, "settings-schema", "s", nil, "Download settings 2.0 objects of one or more settings 2.0 schemas. (Repeat flag or use comma-separated values)")
	cmd.Flags().BoolVar(&f.onlyAPIs, "only-apis", false, "Download only classic configuration APIs. Deprecated configuration APIs will not be included.")
	cmd.Flags().BoolVar(&f.onlySettings, "only-settings", false, "Download only settings 2.0 objects")
	cmd.Flags().BoolVar(&f.onlyAutomation, "only-automation", false, "Only download automation objects, skip all other configuration types")

	// combinations
	cmd.MarkFlagsMutuallyExclusive("settings-schema", "only-apis", "only-settings", "only-automation")
	cmd.MarkFlagsMutuallyExclusive("api", "only-apis", "only-settings", "only-automation")

	if featureflags.Temporary[featureflags.Documents].Enabled() {
		cmd.Flags().BoolVar(&f.onlyDocuments, "only-documents", false, "Only download documents, skip all other configuration types")
		cmd.MarkFlagsMutuallyExclusive("settings-schema", "only-apis", "only-settings", "only-automation", "only-documents")
		cmd.MarkFlagsMutuallyExclusive("api", "only-apis", "only-settings", "only-automation", "only-documents")
	}

	if featureflags.Temporary[featureflags.OpenPipeline].Enabled() {
		cmd.Flags().BoolVar(&f.onlyOpenPipeline, "only-openpipeline", false, "Only download openpipeline configurations, skip all other configuration types")
	}

	if featureflags.Temporary[featureflags.Segments].Enabled() {
		cmd.Flags().BoolVar(&f.onlyGrailFilterSegments, "only-filtersegments", false, "Only download Grail filter-segment configurations, skip all other configuration types")
	}

	err := errors.Join(
		cmd.RegisterFlagCompletionFunc("token", completion.EnvVarName),
		cmd.RegisterFlagCompletionFunc("oauth-client-id", completion.EnvVarName),
		cmd.RegisterFlagCompletionFunc("oauth-client-secret", completion.EnvVarName),

		cmd.RegisterFlagCompletionFunc("manifest", completion.YamlFile),

		cmd.RegisterFlagCompletionFunc("api", completion.AllAvailableApis),
	)

	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return cmd
}

func preRunChecks(f downloadCmdOptions) error {
	switch {
	case f.environmentURL != "" && f.manifestFile != "manifest.yaml":
		return errors.New("'url' and 'manifest' are mutually exclusive")
	case f.environmentURL != "" && f.specificEnvironmentName != "":
		return errors.New("'environment' is specific to manifest-based download and incompatible with direct download from 'url'")
	case f.environmentURL != "":
		switch {
		case f.token == "":
			return errors.New("if 'url' is set, 'token' also must be set")
		case (f.clientID == "") != (f.clientSecret == ""):
			return errors.New("'oauth-client-id' and 'oauth-client-secret' must always be set together")
		default:
			return nil
		}
	case f.manifestFile != "":
		switch {
		case f.token != "" || f.clientID != "" || f.clientSecret != "":
			return errors.New("'token', 'oauth-client-id' and 'oauth-client-secret' can only be used with 'url', while 'manifest' must NOT be set ")
		case f.specificEnvironmentName == "":
			return errors.New("to download with manifest, 'environment' needs to be specified")
		}
	}

	return nil
}

func setupSharedFlags(cmd *cobra.Command, project, outputFolder *string, forceOverwrite *bool) {
	// flags always available
	cmd.Flags().StringVarP(project, "project", "p", "project", "Project to create within the output-folder")
	cmd.Flags().StringVarP(outputFolder, "output-folder", "o", "", "Folder to write downloaded configs to")
	cmd.Flags().BoolVarP(forceOverwrite, "force", "f", false, "Force overwrite any existing manifest.yaml, rather than creating an additional manifest_{timestamp}.yaml. Manifest download: Never append the source environment name to the project folder name.")

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
		httpClient = clientAuth.NewTokenAuthClient(env.Auth.Token.Value.Value())
	} else {
		credentials := clientAuth.OauthCredentials{
			ClientID:     env.Auth.OAuth.ClientID.Value.Value(),
			ClientSecret: env.Auth.OAuth.ClientSecret.Value.Value(),
			TokenURL:     env.Auth.OAuth.GetTokenEndpointValue(),
		}
		httpClient = clientAuth.NewOAuthClient(context.TODO(), credentials)
	}

	url, err := url.Parse(env.URL.Value)
	if err != nil {
		log.Error("Invalid environment URL: %s", err)
		return
	}

	serverVersion, err = versionClient.GetDynatraceVersion(context.TODO(), corerest.NewClient(url, httpClient, corerest.WithRateLimiter(), corerest.WithRetryOptions(&client.DefaultRetryOptions)))
	if err != nil {
		log.WithFields(field.Environment(env.Name, env.Group), field.Error(err)).Warn("Unable to determine server version %q: %v", env.URL.Value, err)
		return
	}
	if serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262}) {
		logUploadToSameEnvironmentWarning()
	}
}

func logUploadToSameEnvironmentWarning() {
	log.Warn("Uploading Settings 2.0 objects to the same environment is not possible due to your cluster version being below '1.262.0'. " +
		"Monaco only reliably supports higher Dynatrace versions for updating downloaded settings without duplicating configurations. " +
		"Consider upgrading to '1.262+'")
}
