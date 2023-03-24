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

package match

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner/completion"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func GetMatchCommand(fs afero.Fs, command Command) (matchCmd *cobra.Command) {

	matchCmd = &cobra.Command{
		Use:     "match <match.yaml>",
		Short:   "Match environments defined in match.yaml from the environments defined in the manifest",
		Example: "monaco match match.yaml",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) >= 2 {
				return fmt.Errorf(`only the match.yaml file can be provided and it is optional`)
			}
			return nil
		},
		PreRun: cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			matchFile := "match.yaml"
			if len(args) >= 1 {
				matchFile = args[0]
			}

			return command.Match(fs, matchFile)
		},
		ValidArgsFunction: completion.MatchCompletion,
	}

	return matchCmd
}
