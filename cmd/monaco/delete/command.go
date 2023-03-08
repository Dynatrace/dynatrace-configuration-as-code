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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func GetDeleteCommand(fs afero.Fs) (deleteCmd *cobra.Command) {

	var environments []string
	var manifestName, group string

	deleteCmd = &cobra.Command{
		Use:     "delete <manifest.yaml> <delete.yaml>",
		Short:   "Delete configurations defined in delete.yaml from the environments defined in the manifest",
		Example: "monaco delete manifest.yaml delete.yaml -e dev-environment",
		Args:    cobra.ExactArgs(2),
		PreRun:  cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName = args[0]
			deleteFile := args[1]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! expected a .yaml file, but got %s", manifestName)
				return err
			}

			if deleteFile != "delete.yaml" {
				err := fmt.Errorf("wrong format for delete file! Has to be named 'delete.yaml', but got %s", deleteFile)
				return err
			}

			return Delete(fs, manifestName, deleteFile, environments, group)
		},
		ValidArgsFunction: completion.DeleteCompletion,
	}

	deleteCmd.Flags().StringVarP(&group, "group", "g", "", "Specify the environmentGroup that should be used for deletion. This flag is mutually exclusive with '--environment'. If this flag is specified, configuration will be deleted from all environments within the specified group.")
	deleteCmd.Flags().StringSliceVarP(&environments, "environment", "e", make([]string, 0), "Deletes configuration only for specified environments. This flag is mutually exclusive with '--group' If not set, delete will be executed on all environments defined in manifest.")

	if err := deleteCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	deleteCmd.MarkFlagsMutuallyExclusive("environment", "group")

	return deleteCmd
}
