/*
 * @license
 * Copyright 2023 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package account

import (
	"log"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
)

type downloadOpts struct {
	auth
	manifestName   string
	accountList    []string
	projectName    string
	accountUUID    string
	outputFolder   string
	forceOverwrite bool
}

type auth struct {
	clientID, clientSecret string
}

func downloadCommand(fs afero.Fs) *cobra.Command {
	opts := downloadOpts{}

	cmd := &cobra.Command{
		Use:               "download [flags]",
		Short:             "Download account management resources",
		Example:           "monaco account download --manifest manifest.yaml --account <account-name-defined-in-manifest> --project <project-defined-in-manifest>",
		ValidArgsFunction: completion.SingleArgumentManifestFileCompletion,
		PreRun:            cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return downloadAll(cmd.Context(), fs, &opts)
		},
	}

	cmd.Flags().StringVarP(&opts.manifestName, "manifest", "m", "manifest.yaml", "Name (and the path) to the manifest file. Defaults to 'manifest.yaml'")
	cmd.Flags().StringSliceVarP(&opts.accountList, "account", "a", []string{}, "List of account names defined in the manifest to download from")
	cmd.Flags().StringVarP(&opts.projectName, "project", "p", "accounts", "Project name defined in the manifest")
	cmd.Flags().StringVarP(&opts.accountUUID, "uuid", "u", "", "Account uuid to use. Required when not using the '--manifest' flag")
	cmd.Flags().StringVar(&opts.clientID, "oauth-client-id", "", "OAuth client ID environment variable. Required when using the '--uuid' flag")
	cmd.Flags().StringVar(&opts.clientSecret, "oauth-client-secret", "", "OAuth client secret environment variable. Required when using the '--uuid' flag")
	cmd.Flags().StringVarP(&opts.outputFolder, "output-folder", "o", "", "Folder to write downloaded resources to")
	cmd.Flags().BoolVarP(&opts.forceOverwrite, "force", "f", false, "Force overwrite any existing manifest.yaml, rather than creating an additional manifest_{timestamp}.yaml. Manifest download: Never append the source environment name to the project folder name")

	cmd.MarkFlagsMutuallyExclusive("manifest", "uuid")
	cmd.MarkFlagsRequiredTogether("uuid", "oauth-client-id", "oauth-client-secret")

	err := cmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatalf("failed to setup CLI %v", err)
	}

	return cmd
}
