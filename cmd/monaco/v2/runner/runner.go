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
	"fmt"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/convert"
	legacyDeploy "github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/v2/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/envvars"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
	"github.com/spf13/afero"

	"github.com/urfave/cli/v2"
)

type ConfigurationError struct {
}

func (c *ConfigurationError) Error() string {
	return ""
}

func Run(args []string) int {
	return RunImpl(args, afero.NewOsFs())
}

func RunImpl(args []string, fs afero.Fs) (statusCode int) {
	var app *cli.App

	app = buildCli(fs)

	err := app.Run(args)

	if err != nil {
		if _, ok := err.(*ConfigurationError); !ok {
			// Log error if it wasn't an ConfigurationError
			util.Log.Error(err.Error())
		}
		return 1
	}

	return 0
}

func buildCli(fs afero.Fs) *cli.App {
	app := cli.NewApp()

	app.Usage = "Automates the deployment of Dynatrace Monitoring Configuration to one or multiple Dynatrace environments."

	app.Version = version.MonitoringAsCode

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Println(c.App.Version)
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:  "version",
		Usage: "print the version",
	}

	app.Description = `
Tool used to deploy dynatrace configurations via the cli

Examples:
  Deploy a manifest
    monaco deploy service.yaml

  Deploy a a specific environment within an manifest
    monaco deploy -s dev service.yaml
`
	var deployCommand cli.Command
	downloadCommand := getDownloadCommand(fs)
	convertCommand := getConvertCommand(fs)

	if isEnvFlagEnabled("CONFIG_V1") {
		util.Log.Warn("CONFIG_V1 environment var detected!")
		util.Log.Warn("Please convert your config to v2 format, as the migration layer will get removed in one of the next releases!")
		deployCommand = getLegacyDeployCommand(fs)
	} else {
		deployCommand = getDeployCommand(fs)
	}

	app.Commands = []*cli.Command{&deployCommand, &convertCommand, &downloadCommand}

	return app
}

func configureLogging(ctx *cli.Context) error {
	err := util.SetupLogging(ctx.Bool("verbose"))

	if err != nil {
		return err
	}

	util.Log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

	return nil
}

func getDeployCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "deploy",
		Usage:     "deploys the given environment",
		UsageText: "deploy [command options] deployment-manifest",
		ArgsUsage: "[working directory]",
		Before:    configureLogging,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.StringFlag{
				Name:    "environment",
				Usage:   "Environment to deploy to",
				Aliases: []string{"e"},
			},
			&cli.StringFlag{
				Name:    "project",
				Usage:   "Project configuration to deploy (also deploys any dependent configurations)",
				Aliases: []string{"p"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Switches to just validation instead of actual deployment",
			},
			&cli.BoolFlag{
				Name:    "continue-on-error",
				Usage:   "Proceed deployment even if config upload fails",
				Aliases: []string{"c"},
			},
		},
		Action: func(ctx *cli.Context) error {
			args := ctx.Args()

			if !args.Present() {
				util.Log.Error("deployment manifest path missing")
				cli.ShowSubcommandHelp(ctx)
				return &ConfigurationError{}
			}

			if args.Len() > 1 {
				util.Log.Error("too many arguments")
				cli.ShowSubcommandHelp(ctx)
				return &ConfigurationError{}
			}

			return deploy.Deploy(
				fs,
				args.First(),
				ctx.String("specific-environment"),
				ctx.String("project"),
				ctx.Bool("dry-run"),
				ctx.Bool("continue-on-error"),
			)
		},
	}
	return command
}

func getConvertCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "convert",
		ArgsUsage: "<path to monaco project folder to convert>",
		Before:    configureLogging,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.PathFlag{
				Name:      "environments",
				Usage:     "Yaml file containing environment to deploy to",
				Aliases:   []string{"e"},
				Required:  true,
				TakesFile: true,
			},
			&cli.PathFlag{
				Name:      "outputFolder",
				Usage:     "Folder where to write converted config to",
				Aliases:   []string{"o"},
				Required:  true,
				TakesFile: false,
			},
			&cli.StringFlag{
				Name:     "manifestName",
				Usage:    "Name of the manifest file to create",
				Required: true,
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.NArg() > 1 {
				util.Log.Error("Too many arguments! Either specify a relative path to the working directory, or omit it for using the current working directory.")
				cli.ShowAppHelpAndExit(ctx, 1)
			}

			var workingDir string

			if ctx.Args().Present() {
				workingDir = ctx.Args().First()
			} else {
				workingDir = "."
			}

			manifestName := ctx.String("manifestName")

			if !strings.HasSuffix(manifestName, ".yaml") {
				manifestName = manifestName + ".yaml"
			}

			return convert.Convert(
				fs,
				workingDir,
				ctx.Path("environments"),
				ctx.Path("outputFolder"),
				manifestName,
			)
		},
	}
	return command
}

func getLegacyDeployCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "deploy",
		Usage:     "deploys the given environment in legacy format",
		UsageText: "deploy-legacy [command options] [working directory]",
		ArgsUsage: "[working directory]",
		Before:    configureLogging,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.PathFlag{
				Name:      "environments",
				Usage:     "Yaml file containing environment to deploy to",
				Aliases:   []string{"e"},
				Required:  true,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:    "specific-environment",
				Usage:   "Specific environment (from list) to deploy to",
				Aliases: []string{"s"},
			},
			&cli.StringFlag{
				Name:    "project",
				Usage:   "Project configuration to deploy (also deploys any dependent configurations)",
				Aliases: []string{"p"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
				Usage:   "Switches to just validation instead of actual deployment",
			},
			&cli.BoolFlag{
				Name:    "continue-on-error",
				Usage:   "Proceed deployment even if config upload fails",
				Aliases: []string{"c"},
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.NArg() > 1 {
				util.Log.Error("Too many arguments! Either specify a relative path to the working directory, or omit it for using the current working directory.")
				cli.ShowAppHelpAndExit(ctx, 1)
			}

			var workingDir string

			if ctx.Args().Present() {
				workingDir = ctx.Args().First()
			} else {
				workingDir = "."
			}

			return legacyDeploy.Deploy(
				fs,
				workingDir,
				ctx.Path("environments"),
				ctx.String("specific-environment"),
				ctx.String("project"),
				ctx.Bool("dry-run"),
				ctx.Bool("continue-on-error"),
			)
		},
	}
	return command
}

func getDownloadCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "download",
		Usage:     "download the given environment",
		UsageText: "download [command options] [working directory]",
		Before:    configureLogging,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
			},
			&cli.PathFlag{
				Name:      "environments",
				Usage:     "Yaml file containing environment to deploy to",
				Aliases:   []string{"e"},
				Required:  true,
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:    "specific-environment",
				Usage:   "Specific environment (from list) to deploy to",
				Aliases: []string{"s"},
			},
			&cli.StringFlag{
				Name:    "downloadSpecificAPI",
				Usage:   "Comma separated list of API's to download ",
				Aliases: []string{"p"},
			},
		},
		Action: func(ctx *cli.Context) error {
			var workingDir string

			if ctx.Args().Present() {
				workingDir = ctx.Args().First()
			} else {
				workingDir = "."
			}

			return download.GetConfigsFilterByEnvironment(
				workingDir,
				fs,
				ctx.Path("environments"),
				ctx.String("specific-environment"),
				ctx.String("downloadSpecificAPI"),
			)
		},
	}
	return command
}

func isEnvFlagEnabled(env string) bool {
	val, ok := envvars.Lookup(env)

	return ok && val != "0"
}
