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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/convert"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/purge"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/support"
	versionCommand "github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/memory"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"io"
)

func RunCmd(fs afero.Fs, command *cobra.Command) int {
	defer func() { // writing the support archive is a deferred function call in order to guarantee that a support archive is also written in case of a panic
		if support.SupportArchive {
			if err := support.Archive(fs); err != nil {
				log.WithFields(field.Error(err)).Error("Encountered error creating support archive. Archive may be missing or incomplete: %s", err)
			}
		}
	}()
	if err := command.Execute(); err != nil {
		log.WithFields(field.Error(err)).Error("Error: %v", err)
		log.WithFields(field.F("errorLogFilePath", log.ErrorFilePath())).Error("error logs written to %s", log.ErrorFilePath())
		return 1
	}
	return 0
}

func BuildCli(fs afero.Fs) *cobra.Command {
	return BuildCliWithLogSpy(fs, nil)
}

func BuildCliWithLogSpy(fs afero.Fs, logSpy io.Writer) *cobra.Command {
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

		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.PrepareLogging(fs, verbose, logSpy, featureflags.LogToFile().Enabled() || support.SupportArchive)

			s := cmd.Name()
			_ = s
			// log the version except for running the main command, help command and version command
			if (cmd.Name() != "monaco") && (cmd.Name() != "help") && (cmd.Name() != "version") {
				version.LogVersionAsInfo()
			}

			memory.SetDefaultLimit()
		},
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
		SilenceErrors: true, // we want to log returned errors on our own, instead of cobra presenting that via println
	}

	// global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&support.SupportArchive, "support-archive", false, "Create support archive")

	// commands
	rootCmd.AddCommand(download.GetDownloadCommand(fs, &download.DefaultCommand{}))
	rootCmd.AddCommand(convert.GetConvertCommand(fs))
	rootCmd.AddCommand(deploy.GetDeployCommand(fs))
	rootCmd.AddCommand(delete.GetDeleteCommand(fs))
	rootCmd.AddCommand(versionCommand.GetVersionCommand())
	rootCmd.AddCommand(generate.Command(fs))

	if featureflags.AccountManagement().Enabled() {
		rootCmd.AddCommand(account.Command(fs))
	}

	if featureflags.DangerousCommands().Enabled() {
		log.Warn("MONACO_ENABLE_DANGEROUS_COMMANDS environment var detected!")
		log.Warn("Use additional commands with care, they might have heavy impact on configurations or environments")

		rootCmd.AddCommand(purge.GetPurgeCommand(fs))
	}

	return rootCmd
}
