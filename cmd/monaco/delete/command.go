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

package delete

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

func GetDeleteCommand(fs afero.Fs) (deleteCmd *cobra.Command) {
	var environments, groups []string
	var manifestName string
	var deleteFile string

	deleteCmd = &cobra.Command{
		Use:     "delete --manifest <manifest.yaml> --file <delete.yaml>",
		Short:   "Delete configurations defined in delete.yaml from the environments defined in the manifest",
		Example: "monaco delete --manifest manifest.yaml --file delete.yaml --environment dev-environment",
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
			manifest, errs := manifestloader.Load(&manifestloader.Context{
				Fs:           fs,
				ManifestPath: absManifestFilePath,
				Environments: environments,
				Groups:       groups,
				Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
			})
			if len(errs) > 0 {
				errutils.PrintErrors(errs)
				return errors.New("error while loading manifest")
			}

			// Try to load delete entries from delete file
			entriesToDelete, err := delete.LoadEntriesFromFile(fs, deleteFile)
			if err != nil {
				return fmt.Errorf("encountered errors while parsing %s: %w", deleteFile, err)
			}

			return Delete(cmd.Context(), manifest.Environments.SelectedEnvironments, entriesToDelete)
		},
		ValidArgsFunction: completion.DeleteCompletion,
	}

	deleteCmd.Flags().StringVarP(&manifestName, "manifest", "m", "manifest.yaml", "The manifest defining the environments to delete from. (default: 'manifest.yaml' in the current folder)")
	deleteCmd.Flags().StringVar(&deleteFile, "file", "delete.yaml", "The delete file defining which configurations to remove. (default: 'delete.yaml' in the current folder)")

	deleteCmd.Flags().StringSliceVarP(&groups, "group", "g", []string{},
		"Specify one (or multiple) environmentGroup(s) that should be used for deletion. "+
			"To set multiple groups either repeat this flag, or separate them using a comma (,). "+
			"This flag is mutually exclusive with '--environment'. "+
			"If this flag is specified, configuration will be deleted from all environments within the specified groups. "+
			"If neither --groups nor --environment is present, all environments will be used for deletion")
	deleteCmd.Flags().StringSliceVarP(&environments, "environment", "e", []string{},
		"Specify one (or multiple) environments(s) that should be used for deletion. "+
			"To set multiple environments either repeat this flag, or separate them using a comma (,). "+
			"This flag is mutually exclusive with '--group'. "+
			"If this flag is specified, configuration will be deleted from all specified environments. "+
			"If neither --groups nor --environment is present, all environments will be used for deletion")

	if err := deleteCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		slog.Error("Failed to set up CLI", log.ErrorAttr(err))
		os.Exit(1)
	}

	deleteCmd.MarkFlagsMutuallyExclusive("environment", "group")

	return deleteCmd
}
