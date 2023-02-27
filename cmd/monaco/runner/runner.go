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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/purge"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"io"

	builtinLog "log"

	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/convert"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/download"
)

var optionalAddedLogger *builtinLog.Logger

func Run() int {
	rootCmd := BuildCli(afero.NewOsFs())

	err := rootCmd.Execute()

	if err != nil {
		log.Error("%v\n", err)
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
	deployCommand := deploy.GetDeployCommand(fs)
	deleteCommand := delete.GetDeleteCommand(fs)
	purgeCommand := purge.GetPurgeCommand(fs)
	versionCommand := version.GetVersionCommand()

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
