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

package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/convert"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/delete"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/download"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner/completion"
	legacyDeploy "github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/v1/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
	builtinLog "log"
)

var errWrongUsage = errors.New("")

var optionalAddedLogger *builtinLog.Logger

func Run() int {
	rootCmd := BuildCli(afero.NewOsFs())

	err := rootCmd.Execute()

	if err != nil {
		if !errors.Is(err, errWrongUsage) {
			// Log error if it wasn't a usage error
			log.Error("%v\n", err)
		}
		return 1
	}

	return 0
}

func BuildCliWithCapturedLog(fs afero.Fs, logOutput io.Writer) *cobra.Command {
	optionalAddedLogger = builtinLog.New(logOutput, "", builtinLog.LstdFlags)

	cmd := BuildCli(fs)
	return cmd
}

func BuildCli(fs afero.Fs) *cobra.Command {
	var verbose bool

	var rootCmd = &cobra.Command{
		Use:   "monaco <command>",
		Short: "Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments.",
		Long: `Tool used to deploy dynatrace configurations via the cli

Examples:
  Deploy configuration defined in a manifest
    monaco deploy service.yaml
  Deploy a specific environment within an manifest
    monaco deploy service.yaml -e dev`,

		PersistentPreRunE: configureDebugLogging(&verbose),
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")

	// commands
	downloadCommand := download.GetDownloadCommand(fs, &download.DefaultCommand{})
	convertCommand := getConvertCommand(fs)
	deployCommand := getDeployCommand(fs)
	deleteCommand := getDeleteCommand(fs)
	purgeCommand := getPurgeCommand(fs)

	if isEnvFlagEnabled("CONFIG_V1") {
		log.Warn("CONFIG_V1 environment var detected!")
		log.Warn("Please convert your config to v2 format, as the migration layer will get removed in one of the next releases!")
		deployCommand = getLegacyDeployCommand(fs)
	}

	rootCmd.AddCommand(downloadCommand)
	rootCmd.AddCommand(convertCommand)
	rootCmd.AddCommand(deployCommand)
	rootCmd.AddCommand(deleteCommand)

	if isEnvFlagEnabled("MONACO_ENABLE_DANGEROUS_COMMANDS") {
		log.Warn("MONACO_ENABLE_DANGEROUS_COMMANDS environment var detected!")
		log.Warn("Use additional commands with care, they might have heavy impact on configurations or environments")

		rootCmd.AddCommand(purgeCommand)
	}

	return rootCmd
}

func configureDebugLogging(verbose *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if *verbose {
			log.Default().SetLevel(log.LevelDebug)
		}

		err := log.SetupLogging(optionalAddedLogger)
		if err != nil {
			return err
		}

		log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

		return nil
	}
}

func getDeployCommand(fs afero.Fs) (deployCmd *cobra.Command) {
	var dryRun, continueOnError bool
	var manifestName string
	var environment, project []string

	deployCmd = &cobra.Command{
		Use:               "deploy <manifest.yaml>",
		Short:             "Deploy configurations to Dynatrace environments",
		Example:           "monaco deploy manifest.yaml -v -e dev-environment",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.DeployCompletion,
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName = args[0]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! expected a .yaml file, but got %s", manifestName)
				return err
			}

			return deploy.Deploy(fs, manifestName, environment, project, dryRun, continueOnError)
		},
	}

	deployCmd.Flags().StringSliceVarP(&environment, "environment", "e", make([]string, 0), "Environment to deploy to")
	deployCmd.Flags().StringSliceVarP(&project, "project", "p", make([]string, 0), "Project configuration to deploy (also deploys any dependent configurations)")
	deployCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Switches to just validation instead of actual deployment")
	deployCmd.Flags().BoolVarP(&continueOnError, "continue-on-error", "c", false, "Proceed deployment even if config upload fails")
	err := deployCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByManifestFlag)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	err = deployCmd.RegisterFlagCompletionFunc("project", completion.ProjectsFromManifest)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	return deployCmd
}

func getDeleteCommand(fs afero.Fs) (deleteCmd *cobra.Command) {

	var environment []string
	var manifestName string

	deleteCmd = &cobra.Command{
		Use:     "delete <manifest.yaml> <delete.yaml>",
		Short:   "Delete configurations defined in delete.yaml from the environments defined in the manifest",
		Example: "monaco delete manifest.yaml delete.yaml -e dev-environment",
		Args:    cobra.ExactArgs(2),
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
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

			return delete.Delete(fs, manifestName, deleteFile, environment)
		},
		ValidArgsFunction: completion.DeleteCompletion,
	}

	deleteCmd.Flags().StringSliceVarP(&environment, "environment", "e", make([]string, 0), "Deletes configuration only for specified envs. If not set, delete will be executed on all environments defined in manifest.")

	if err := deleteCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return deleteCmd
}

func getPurgeCommand(fs afero.Fs) (purgeCmd *cobra.Command) {

	var environment []string
	var manifestName string
	var specificApis []string

	purgeCmd = &cobra.Command{
		Use:     "purge <manifest.yaml>",
		Short:   "Delete ALL configurations from the environments defined in the manifest",
		Example: "monaco purge manifest.yaml -e dev-environment",
		Hidden:  true, // this command will not be suggested or shown in help
		Args:    cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName = args[0]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! expected a .yaml file, but got %s", manifestName)
				return err
			}

			return delete.Purge(fs, manifestName, environment, specificApis)
		},
		ValidArgsFunction: completion.PurgeCompletion,
	}

	purgeCmd.Flags().StringSliceVarP(&environment, "environment", "e", make([]string, 0), "Deletes configuration only for specified envs. If not set, delete will be executed on all environments defined in manifest.")
	purgeCmd.Flags().StringSliceVarP(&specificApis, "specific-api", "a", make([]string, 0), "APIs to purge")

	if err := purgeCmd.RegisterFlagCompletionFunc("environment", completion.EnvironmentByArg0); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	if err := purgeCmd.RegisterFlagCompletionFunc("specific-api", completion.AllAvailableApis); err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}

	return purgeCmd
}

func getConvertCommand(fs afero.Fs) (convertCmd *cobra.Command) {

	var outputFolder, manifestName string

	convertCmd = &cobra.Command{
		Use:               "convert <environment.yaml> <config folder to convert>",
		Short:             "Convert v1 monaco configuration into v2 format",
		Example:           "monaco convert environment.yaml my-v1-project -o my-v2-project",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completion.ConvertCompletion,
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			environmentsFile := args[0]
			workingDir := args[1]

			if !files.IsYamlFileExtension(environmentsFile) {
				err := fmt.Errorf("wrong format for environment file! expected a .yaml file, but got %s", environmentsFile)
				return err
			}

			if !files.IsYamlFileExtension(manifestName) {
				manifestName = manifestName + ".yaml"
			}

			if outputFolder == "{project folder}-v2" {
				outputFolder = filepath.Clean(workingDir) + "-v2"
			}

			return convert.Convert(fs, workingDir, environmentsFile, outputFolder, manifestName)
		},
	}

	convertCmd.Flags().StringVarP(&manifestName, "manifest", "m", "manifest.yaml", "Name of the manifest file to create")
	convertCmd.Flags().StringVarP(&outputFolder, "output-folder", "o", "{project folder}-v2", "Folder where to write converted config to")
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

func getLegacyDeployCommand(fs afero.Fs) (deployCmd *cobra.Command) {
	var dryRun, continueOnError bool
	var specificEnvironment, environments, projects string

	deployCmd = &cobra.Command{
		Use:     "deploy [configuration directory]",
		Short:   "Deploy v1 configurations to Dynatrace environments",
		Example: "monaco deploy -e environments.yaml",
		PreRun: func(cmd *cobra.Command, args []string) {
			cmd.SilenceUsage = true
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) > 1 {
				log.Error("too many arguments")
				return errWrongUsage
			}
			workingDir := "."
			if len(args) != 0 {
				workingDir = args[0]
			}

			return legacyDeploy.Deploy(fs, workingDir, environments, specificEnvironment, projects, dryRun, continueOnError)
		},
	}

	deployCmd.Flags().StringVarP(&environments, "environments", "e", "", "Yaml file containing environment to deploy to")
	deployCmd.Flags().StringVarP(&projects, "project", "p", "", "Project configuration to deploy (also deploys any dependent configurations)")
	deployCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Switches to just validation instead of actual deployment")
	deployCmd.Flags().BoolVarP(&continueOnError, "continue-on-error", "c", false, "Proceed deployment even if config upload fails")
	err := deployCmd.MarkFlagFilename("environments", files.YamlExtensions...)
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	err = deployCmd.MarkFlagRequired("environments")
	if err != nil {
		log.Fatal("failed to setup CLI %v", err)
	}
	return deployCmd
}

func isEnvFlagEnabled(env string) bool {
	val, ok := os.LookupEnv(env)
	return ok && val != "0"
}
