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

package generate

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/deletefile"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/dependencygraph"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate/schemas"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Command(fs afero.Fs) (cmd *cobra.Command) {

	cmd = &cobra.Command{
		Use:     "generate",
		Short:   "Generate offers several sub-commands to generate files - take a look at the sub-commands for usage",
		Example: "monaco generate graph --manifest manifest.yaml -e dev-environment -o mygraphs_folder",
		Args:    cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(dependencygraph.Command(fs))
	cmd.AddCommand(deletefile.Command(fs))
	cmd.AddCommand(schemas.Command(fs))

	return cmd
}
