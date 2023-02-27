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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/purge"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"io"

	builtinLog "log"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/convert"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner/completion"
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

		PersistentPreRun: configureDebugLogging(fs, &verbose),
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")

	// commands
	downloadCommand := download.GetDownloadCommand(fs, &download.DefaultCommand{})
	convertCommand := convert.GetConvertCommand(fs)
	deployCommand := getDeployCommand(fs)
	deleteCommand := getDeleteCommand(fs)
	purgeCommand := purge.GetPurgeCommand(fs)
	versionCommand := getVersionCommand()

	rootCmd.AddCommand(downloadCommand)
	rootCmd.AddCommand(convertCommand)
	rootCmd.AddCommand(deployCommand)
	rootCmd.AddCommand(deleteCommand)
	rootCmd.AddCommand(versionCommand)

	if featureflags.FeatureFlagEnabled("MONACO_ENABLE_DANGEROUS_COMMANDS") {
		log.Warn("MONACO_ENABLE_DANGEROUS_COMMANDS environment var detected!")
		log.Warn("Use additional commands with care, they might have heavy impact on configurations or environments")

		rootCmd.AddCommand(purgeCommand)
	}

	return rootCmd
}

func configureDebugLogging(fs afero.Fs, verbose *bool) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if *verbose {
			log.Default().SetLevel(log.LevelDebug)
		}
		log.SetupLogging(fs, optionalAddedLogger)
	}
}

func getDeployCommand(fs afero.Fs) (deployCmd *cobra.Command) {
	var dryRun, continueOnError bool
	var manifestName, group string
	var environment, project []string

	deployCmd = &cobra.Command{
		Use:               "deploy <manifest.yaml>",
		Short:             "Deploy configurations to Dynatrace environments",
		Example:           "monaco deploy manifest.yaml -v -e dev-environment",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.DeployCompletion,
		PreRun:            cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {

			manifestName = args[0]

			if !files.IsYamlFileExtension(manifestName) {
				err := fmt.Errorf("wrong format for manifest file! expected a .yaml file, but got %s", manifestName)
				return err
			}

			return deploy.Deploy(fs, manifestName, environment, group, project, dryRun, continueOnError)
		},
	}

	deployCmd.Flags().StringSliceVarP(&environment, "environment", "e", make([]string, 0), "Specify one (or multiple) environments to deploy to. To set multiple environments either repeat this flag, or seperate them using a comma (,). This flag is mutually exclusive with '--group'.")
	deployCmd.Flags().StringVarP(&group, "group", "g", "", "Specify the environmentGroup that should be used for deployment. If this flag is specified, all environments within this group will be used for deployment. This flag is mutually exclusive with '--environment'")
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

	deployCmd.MarkFlagsMutuallyExclusive("environment", "group")

	return deployCmd
}

func getDeleteCommand(fs afero.Fs) (deleteCmd *cobra.Command) {

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

			return delete.Delete(fs, manifestName, deleteFile, environments, group)
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

func getVersionCommand() (convertCmd *cobra.Command) {
	versionCmd := &cobra.Command{
		Use:     "version",
		Short:   "Prints out the version of the monaco cli",
		Example: "monaco version",
		PreRun:  cmdutils.SilenceUsageCommand(),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("monaco version " + version.MonitoringAsCode)
		},
	}
	return versionCmd
}
