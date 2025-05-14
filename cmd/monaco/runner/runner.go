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
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/account"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/delete"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/download"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/generate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/purge"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
	versionCommand "github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/memory"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/trafficlogs"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/version"
)

func RunCmd(ctx context.Context, cmd *cobra.Command) error {
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		log.WithFields(field.Error(err)).Error("Error: %v", err)
		log.WithFields(field.F("errorLogFilePath", log.ErrorFilePath())).Error("error logs written to %s", log.ErrorFilePath())
	}
	return err
}

func BuildCmd(fs afero.Fs) *cobra.Command {
	return BuildCmdWithLogSpy(fs, nil)
}

func writeSupportArchive(fs afero.Fs) func() {
	return func() {
		if err := trafficlogs.GetInstance().Sync(); err != nil {
			log.WithFields(field.Error(err)).Error("Encountered error while syncing/flushing traffic log files: %s", err)
		}
		if err := supportarchive.Write(fs); err != nil {
			log.WithFields(field.Error(err)).Error("Encountered error creating support archive. Archive may be missing or incomplete: %s", err)
		}
	}
}

func BuildCmdWithLogSpy(fs afero.Fs, logSpy io.Writer) *cobra.Command {
	var verbose bool
	var supportArchive bool

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
			if supportArchive {
				cobra.OnFinalize(writeSupportArchive(fs))
				cmd.SetContext(supportarchive.ContextWithSupportArchive(cmd.Context()))
			}
			ctx := context.WithValue(cmd.Context(), oauth2.HTTPClient, &http.Client{Transport: &http.Transport{
				TLSNextProto:        make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
				TLSHandshakeTimeout: 5 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,  // Timeout for establishing TCP connection
					KeepAlive: 30 * time.Second, // Keep-alive period for TCP connection
				}).DialContext,
				IdleConnTimeout:       90 * time.Second, // How long idle connections stay in the pool
				ExpectContinueTimeout: 1 * time.Second,  // Wait time for 100-continue response
				MaxIdleConns:          100,              // Max idle connections across all hosts
				MaxIdleConnsPerHost:   10,               // Max idle connections per host
			}})
			cmd.SetContext(ctx)
			fileBasedLogging := featureflags.LogToFile.Enabled() || supportArchive
			memStatLogging := featureflags.LogMemStats.Enabled()
			log.PrepareLogging(cmd.Context(), fs, verbose, logSpy, fileBasedLogging, memStatLogging)

			// log the version except for running the main command, help command and version command
			if (cmd.Name() != "monaco") && (cmd.Name() != "help") && (cmd.Name() != "version") {
				version.LogVersionAsInfo()
			}

			if featureflags.AnyModified() {
				log.Warn("Feature Flags modified - Dynatrace Support might not be able to assist you with issues.")
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
	rootCmd.PersistentFlags().BoolVar(&supportArchive, "support-archive", false, "Create support archive")

	// commands
	rootCmd.AddCommand(download.GetDownloadCommand(fs, &download.DefaultCommand{}))
	rootCmd.AddCommand(deploy.GetDeployCommand(fs))
	rootCmd.AddCommand(delete.GetDeleteCommand(fs))
	rootCmd.AddCommand(versionCommand.GetVersionCommand())
	rootCmd.AddCommand(generate.Command(fs))

	rootCmd.AddCommand(account.Command(fs))

	if featureflags.DangerousCommands.Enabled() {
		log.Warn("MONACO_ENABLE_DANGEROUS_COMMANDS environment var detected!")
		log.Warn("Use additional commands with care, they might have heavy impact on configurations or environments")

		rootCmd.AddCommand(purge.GetPurgeCommand(fs))
	}

	return rootCmd
}
