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

package deletefile

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

func Command(fs afero.Fs) (cmd *cobra.Command) {

	var fileName, outputFolder string
	var projects, environments []string
	var includeTypes, excludeTypes []string

	cmd = &cobra.Command{
		Use:               "deletefile <manifest.yaml>",
		Short:             "Generate a delete file for all configurations defined in the given manifest's projects",
		Example:           "monaco generate deletefile manifest.yaml -o deletefiles --file my-projects-delete-file.yaml",
		Args:              cobra.ExactArgs(1),
		PreRun:            cmdutils.SilenceUsageCommand(),
		ValidArgsFunction: completion.SingleArgumentManifestFileCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName := args[0]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! Expected a .yaml file, but got %s", manifestName)
				return err
			}

			m, errs := manifestloader.Load(&manifestloader.Context{
				Fs:           fs,
				ManifestPath: manifestName,
				Opts: manifestloader.Options{
					DoNotResolveEnvironmentGroupEnvVars: true,
					DoNotResolveAccountEnvVars:          true,
					RequireEnvironmentGroups:            true,
				},
			})
			if len(errs) > 0 {
				errutils.PrintErrors(errs)
				return fmt.Errorf("failed to load manifest %q", manifestName)
			}

			apis := api.NewAPIs().Filter(api.RemoveDisabled)
			loadedProjects, errs := project.LoadProjects(cmd.Context(), fs, project.ProjectLoaderContext{
				KnownApis:       apis.GetApiNameLookup(),
				WorkingDir:      filepath.Dir(manifestName),
				Manifest:        m,
				ParametersSerde: config.DefaultParameterParsers,
			}, projects)

			if len(errs) > 0 {
				errutils.PrintErrors(errs)
				return fmt.Errorf("failed to load projects")
			}

			options := createDeleteFileOptions{
				environmentNames: environments,
				fileName:         fileName,
				includeTypes:     includeTypes,
				excludeTypes:     excludeTypes,
				outputFolder:     outputFolder,
			}

			// dashboard-share-settings and OpenPipeline configurations are excluded per default, as they cannot be deleted,
			// hence it makes no sense to generate delete entries for it
			options.excludeTypes = append(options.excludeTypes, api.DashboardShareSettings, string(config.OpenPipelineTypeID))

			return createDeleteFile(fs, loadedProjects, apis, options)
		},
	}

	cmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "", "The folder the generated delete file should be written to. If not set, files will be created in the current directory.")
	cmd.Flags().StringVarP(&fileName, "file", "", "delete.yaml", "The name of the generated delete file. If a file of this name already exists, a timestamp will be appended.")

	cmd.Flags().StringSliceVarP(&projects, "project", "p", nil, "Projects to generate delete file entries for. If not defined, all projects in the manifest will be used.")
	cmd.Flags().StringSliceVar(&excludeTypes, "exclude-types", nil, "Comma-separated list of config types to be excluded from the generation process.")
	cmd.Flags().StringSliceVar(&includeTypes, "types", nil, "Comma-separated list of config types to be included in the generation process.")

	cmd.Flags().StringSliceVarP(&environments, "environment", "e", []string{},
		"Specify one (or multiple) environment(s) to generate delete entries for. If not defined, entries for all environments will be generated. It is generally safe and recommended to generate a full delete file for all environments, but you may sometimes want to create a file limited to a specific environment's overrides.")

	if err := cmd.RegisterFlagCompletionFunc("project", completion.ProjectsFromManifest); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return cmd
}
