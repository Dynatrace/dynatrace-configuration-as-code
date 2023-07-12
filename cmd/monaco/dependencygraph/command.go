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

package dependencygraph

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Command(fs afero.Fs) (cmd *cobra.Command) {

	var environments, groups []string
	var manifestName, outputFolder string

	cmd = &cobra.Command{
		Use:     "graph --manifest <manifest.yaml>",
		Short:   "Generate dependency graphs as DOT/graphviz file per environment for the configurations defined in the manifest.",
		Example: "monaco graph --manifest manifest.yaml -e dev-environment -o mygraphs_folder",
		Args:    cobra.NoArgs,
		PreRun:  cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! Expected a .yaml file, but got %s", manifestName)
				return err
			}

			return writeGraphFiles(fs, manifestName, environments, groups, outputFolder)
		},
		ValidArgsFunction: completion.DeleteCompletion,
	}

	cmd.Flags().StringVarP(&manifestName, "manifest", "m", "manifest.yaml", "The manifest defining the environments and configurations to create dependency graphs for. (default: 'manifest.yaml' in the current folder)")

	cmd.Flags().StringSliceVarP(&groups, "group", "g", []string{},
		"Specify one (or multiple) environmentGroup(s) that should be used for creating depy graphs. "+
			"To set multiple groups either repeat this flag, or separate them using a comma (,). "+
			"This flag is mutually exclusive with '--environment'. "+
			"If this flag is specified, configuration will be deleted from all environments within the specified groups. "+
			"If neither --groups nor --environment is present, all environments will be used for deletion")
	cmd.Flags().StringSliceVarP(&environments, "environment", "e", []string{},
		"Specify one (or multiple) environments(s) that should be used for creating dependency graphs. "+
			"To set multiple environments either repeat this flag, or separate them using a comma (,). "+
			"This flag is mutually exclusive with '--group'. "+
			"If this flag is specified, configuration will be deleted from all specified environments. "+
			"If neither --groups nor --environment is present, all environments will be used for deletion")

	cmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "", "The folder generated dependency graph DOT files should be written to. If not set files will be created in the current directory.")

	if err := cmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	cmd.MarkFlagsMutuallyExclusive("environment", "group")

	return cmd
}
