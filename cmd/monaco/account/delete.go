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

package account

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/clientcredentials"
	"path/filepath"
)

func deleteCommand(fs afero.Fs) *cobra.Command {
	var accounts []string
	var manifestName string
	var deleteFile string

	deleteCmd := &cobra.Command{
		Use:     "delete --manifest <manifest.yaml> --file <delete.yaml>",
		Short:   "Delete account resources defined in delete.yaml from the accounts defined in the manifest",
		Example: "monaco delete --manifest manifest.yaml --file delete.yaml -a dev-account",
		Args:    cobra.NoArgs,
		PreRun:  cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! Expected a .yaml file, but got %s", manifestName)
				return err
			}

			if !files.IsYamlFileExtension(deleteFile) {
				err := fmt.Errorf("wrong format for delete file! Expected a .yaml file, but got %s", deleteFile)
				return err
			}

			// Sanitize manifest file path to manifest yaml file
			manifestName = filepath.Clean(manifestName)
			absManifestFilePath, err := filepath.Abs(manifestName)
			if err != nil {
				return err
			}

			// Try to load the manifest file
			m, errs := manifestloader.Load(&manifestloader.Context{
				Fs:           fs,
				ManifestPath: absManifestFilePath,
			})
			if len(errs) > 0 {
				errutils.PrintErrors(errs)
				return fmt.Errorf("error while loading manifest (%s)", absManifestFilePath)
			}

			// Try to load delete entries from delete file
			entriesToDelete, err := delete.LoadResourcesToDelete(fs, deleteFile)
			if err != nil {
				return fmt.Errorf("failed to parse delete file (%s): %s", deleteFile, err)
			}

			if len(accounts) == 0 {
				accounts = maps.Keys(m.Accounts)
			}
			var errOccurred bool
			for _, name := range accounts {
				account, found := m.Accounts[name]
				if !found {
					log.Error("Account %q is not defined in manifest", name)
					errOccurred = true
				}

				c, err := createClient(account)
				if err != nil {
					log.Error("Failed to create API client for account %q: %v", name, err)
					errOccurred = true
				}
				err = delete.AccountResources(context.Background(), c, account.AccountUUID.String(), entriesToDelete)
				if err != nil {
					log.Error("Failed to delete resources for account %q", name)
					errOccurred = true
				}
			}
			if errOccurred {
				return fmt.Errorf("encountered errors deleting account resoruces - please see logs")
			}
			return nil
		},
		ValidArgsFunction: completion.DeleteCompletion,
	}

	deleteCmd.Flags().StringVarP(&manifestName, "manifest", "m", "manifest.yaml", "The manifest defining the environments to delete from. (default: 'manifest.yaml' in the current folder)")
	deleteCmd.Flags().StringVar(&deleteFile, "file", "delete.yaml", "The delete file defining which configurations to remove. (default: 'delete.yaml' in the current folder)")

	deleteCmd.Flags().StringSliceVarP(&accounts, "account", "a", []string{},
		"Specify one (or multiple) accounts(s) that should be used for deletion. "+
			"To set multiple accounts either repeat this flag, or separate them using a comma (,). "+
			"If this flag is specified, resources will be deleted from all specified accounts. "+
			"If it is not specified, all accounts in the manfiest will be used for deletion")

	if err := deleteCmd.RegisterFlagCompletionFunc("account", completion.AccountsByManifestFlag); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return deleteCmd
}

func createClient(a manifest.Account) (delete.Client, error) {
	oauthCreds := clientcredentials.Config{
		ClientID:     a.OAuth.ClientID.Value.Value(),
		ClientSecret: a.OAuth.ClientSecret.Value.Value(),
		TokenURL:     a.OAuth.GetTokenEndpointValue(),
	}

	var apiUrl string
	if a.ApiUrl == nil || a.ApiUrl.Value == "" {
		apiUrl = "https://api.dynatrace.com"
	} else {
		apiUrl = a.ApiUrl.Value
	}

	c, err := clients.Factory().WithOAuthCredentials(oauthCreds).AccountClient(apiUrl)
	if err != nil {
		return nil, err
	}
	return &delete.AccountAPIClient{Client: c}, nil
}
