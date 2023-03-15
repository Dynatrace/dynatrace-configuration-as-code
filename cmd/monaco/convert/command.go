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

package convert

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"os"
	"path"
)

func GetConvertCommand(fs afero.Fs) (convertCmd *cobra.Command) {

	var outputFolder, manifestName string

	convertCmd = &cobra.Command{
		Use:               "convert <environment.yaml> <config folder to convert>",
		Short:             "Convert v1 monaco configuration into v2 format",
		Example:           "monaco convert environment.yaml my-v1-project -o my-v2-project",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completion.ConvertCompletion,
		PreRun:            cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			environmentsFile := args[0]
			workingDir := args[1]

			if !files.IsYamlFileExtension(environmentsFile) {
				err := fmt.Errorf("wrong format for environment file! expected a .yaml file, but got %s", environmentsFile)
				return err
			}

			if !files.IsYamlFileExtension(manifestName) {
				manifestName += ".yaml"
			}

			if outputFolder == "" {
				folder, err := os.Getwd()
				if err != nil {
					return err
				}

				outputFolder = path.Base(folder) + "-v2"
			}

			return convert(fs, workingDir, environmentsFile, outputFolder, manifestName)
		},
	}

	convertCmd.Flags().StringVarP(&manifestName, "manifest", "m", "manifest.yaml", "Name of the manifest file to create")
	convertCmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "", "Folder where to write converted config to")
	err := convertCmd.MarkFlagDirname("output-folder")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	err = convertCmd.MarkFlagFilename("manifest", files.YamlExtensions...)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	return convertCmd
}
