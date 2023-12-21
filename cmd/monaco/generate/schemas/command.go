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

package schemas

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func Command(fs afero.Fs) (cmd *cobra.Command) {

	var outputFolder string

	cmd = &cobra.Command{
		Use:     "schemas",
		Short:   "Generate JSON schemas for YAML files like manifests, as well as Error types",
		Example: "monaco generate schemas -o output-folder",
		Args:    cobra.NoArgs,
		PreRun:  cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			return generateSchemaFiles(fs, outputFolder)
		},
	}

	cmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "schemas", "The folder the generated schema files should be written to. If not set, files will be created in a 'schemas' folder.")

	return cmd
}
