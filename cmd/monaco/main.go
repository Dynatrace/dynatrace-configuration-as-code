/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package main

import (
	"fmt"
	"os"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/deploy"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/version"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v2"
)

func main() {
	statusCode := Run(os.Args)
	os.Exit(statusCode)
}

func Run(args []string) int {
	return RunImpl(args, afero.NewOsFs())
}

func RunImpl(args []string, fs afero.Fs) (statusCode int) {
	var app *cli.App

	if newCli, ok := os.LookupEnv("NEW_CLI"); ok && newCli != "0" {
		app = buildExperimentalCli(fs)
	} else {
		app = buildCli(fs)
	}

	err := app.Run(args)

	if err != nil {
		util.Log.Error("%s\n", err)
		return 1
	}

	return 0
}

func buildCli(fs afero.Fs) *cli.App {
	fmt.Print(`You are currently using the old CLI structure which will be used by
default until monaco version 2.0.0

Check out the beta of the new CLI by adding the environment variable
  "NEW_CLI".

We can't wait for your feedback.

`)

	app := cli.NewApp()

	app.ArgsUsage = "[working directory]"

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
  Deploy a specific project inside a root config folder:
    monaco -p='project-folder' -e='environments.yaml' projects-root-folder

  Deploy a specific project to a specific tenant:
    monaco --environments environments.yaml --specific-environment dev --project myProject
`

	app.Before = func(c *cli.Context) error {
		err := util.SetupLogging(c.Bool("verbose"))

		if err != nil {
			return err
		}

		util.Log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

		return nil
	}

	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:    "verbose",
			Aliases: []string{"v"},
		},
		&cli.PathFlag{
			Name:      "environments",
			Usage:     "Yaml file containing environments to deploy to",
			Aliases:   []string{"e"},
			Required:  true,
			TakesFile: true,
		},
		&cli.StringFlag{
			Name:        "specific-environment",
			Usage:       "Specific environment (from list) to deploy to",
			Aliases:     []string{"se"},
			DefaultText: "none",
		},
		&cli.StringFlag{
			Name:        "project",
			Usage:       "Project configuration to deploy (also deploys any dependent configurations)",
			Aliases:     []string{"p"},
			DefaultText: "none",
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
	}

	app.Action = func(ctx *cli.Context) error {
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

		deploymentHandler, err := deploy.NewHandler(
			workingDir,
			fs,
			ctx.Path("environments"),
		)
		if err != nil {
			return err
		}

		err = deploymentHandler.RunAll(
			ctx.String("specific-environment"),
			ctx.String("project"),
			ctx.Bool("dry-run"),
			ctx.Bool("continue-on-error"),
		)
		if err != nil {
			return err
		}

		return nil
	}

	return app
}

func buildExperimentalCli(fs afero.Fs) *cli.App {
	fmt.Print(`You are using the new CLI structure which is currently in Beta.

Please provide feedback here:
  https://github.com/dynatrace-oss/dynatrace-monitoring-as-code/issues/45.

We plan to make this CLI GA in version 2.0.0

`)

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
  Deploy a specific project inside a root config folder:
    monaco deploy -p='project-folder' -e='environments.yaml' projects-root-folder

  Deploy a specific project to a specific tenant:
    monaco deploy --environments environments.yaml --specific-environment dev --project myProject
`
	deployCommand := getDeployCommand(fs)
	downloadCommand := getDownloadCommand(fs)
	app.Commands = []*cli.Command{&deployCommand, &downloadCommand}

	return app
}
func getDeployCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "deploy",
		Usage:     "deploys the given environment",
		UsageText: "deploy [command options] [working directory]",
		ArgsUsage: "[working directory]",
		Before: func(c *cli.Context) error {
			err := util.SetupLogging(c.Bool("verbose"))

			if err != nil {
				return err
			}

			util.Log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

			return nil
		},
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

			deploymentHandler, err := deploy.NewHandler(
				workingDir,
				fs,
				ctx.Path("environments"),
			)
			if err != nil {
				return err
			}

			err = deploymentHandler.RunAll(
				ctx.String("specific-environment"),
				ctx.String("project"),
				ctx.Bool("dry-run"),
				ctx.Bool("continue-on-error"),
			)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return command
}
func getDownloadCommand(fs afero.Fs) cli.Command {
	command := cli.Command{
		Name:      "download",
		Usage:     "download the given environment",
		UsageText: "download [command options] [working directory]",
		Before: func(c *cli.Context) error {
			err := util.SetupLogging(c.Bool("verbose"))

			if err != nil {
				return err
			}

			util.Log.Info("Dynatrace Monitoring as Code v" + version.MonitoringAsCode)

			return nil
		},
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
