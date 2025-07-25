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

package purge

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

func GetPurgeCommand(fs afero.Fs) (purgeCmd *cobra.Command) {

	var environment []string
	var manifestName string
	var specificApis []string

	purgeCmd = &cobra.Command{
		Use:     "purge <manifest.yaml>",
		Short:   "Delete ALL configurations from the environments defined in the manifest",
		Example: "monaco purge manifest.yaml -e dev-environment",
		Hidden:  true, // this command will not be suggested or shown in help
		Args:    cobra.ExactArgs(1),
		PreRun:  cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName = args[0]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! expected a .yaml file, but got %s", manifestName)
				return err
			}

			return purge(cmd.Context(), fs, manifestName, environment, specificApis)
		},
		ValidArgsFunction: completion.PurgeCompletion,
	}

	purgeCmd.Flags().StringSliceVarP(&environment, "environment", "e", make([]string, 0), "Deletes configuration only for specified environments. All environments are included if this property is not set. ")
	purgeCmd.Flags().StringSliceVarP(&specificApis, "api", "a", make([]string, 0), "One or more specific APIs to delete from (flag can be repeated or value defined as comma-separated list)")

	if err := purgeCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		slog.Error("Failed to set up CLI", log.ErrorAttr(err))
		os.Exit(1)
	}
	if err := purgeCmd.RegisterFlagCompletionFunc("api", completion.AllAvailableApis); err != nil {
		slog.Error("Failed to set up CLI", log.ErrorAttr(err))
		os.Exit(1)
	}

	return purgeCmd
}
