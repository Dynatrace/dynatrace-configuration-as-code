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

package account

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/environment"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/loader"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

type deployOpts struct {
	workingDir   string
	manifestName string
	accountName  string
	project      string
	dryRun       bool
}

func deployCommand(fs afero.Fs) *cobra.Command {
	opts := deployOpts{}

	command := &cobra.Command{
		Use:               "deploy [flags]",
		Short:             "Deploy account management resources",
		Example:           "monaco account deploy --manifest <path_to_manifest> --account <account-name> --project <project-name>",
		ValidArgsFunction: completion.SingleArgumentManifestFileCompletion,
		PreRun:            cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !files.IsYamlFileExtension(opts.manifestName) {
				return fmt.Errorf("expected a .yaml file, but got %s", opts.manifestName)
			}

			opts.workingDir = filepath.Dir(opts.manifestName)

			return deploy(cmd.Context(), fs, opts)
		},
	}

	command.Flags().StringVarP(&opts.manifestName, "manifest", "m", "manifest.yaml", "Name (and the path) to the manifest file. Defaults to 'manifest.yaml'")
	command.Flags().StringVarP(&opts.accountName, "account", "a", "", "Account name defined in the manifest to deploy to.")
	command.Flags().StringVarP(&opts.project, "project", "p", "", "Project name defined in the manifest")
	command.Flags().BoolVarP(&opts.dryRun, "dry-run", "d", false, "Validate the structure of your manifest, projects and configurations. Dry-run will resolve all configuration parameters but cannot verify if the content will be accepted by the Dynatrace APIs.")

	return command
}

func deploy(ctx context.Context, fs afero.Fs, opts deployOpts) error {
	selectedAccounts := make([]string, 0)
	if opts.accountName != "" {
		selectedAccounts = append(selectedAccounts, opts.accountName)
	}

	mani, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: opts.manifestName,
		Accounts:     selectedAccounts,
		Opts:         manifestloader.Options{RequireAccounts: true},
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return errors.New("error while loading manifest")
	}

	// filter project
	projects := mani.Projects
	if opts.project != "" {
		proj, ok := projects[opts.project]
		if !ok {
			return fmt.Errorf("required project %q was not found in manifest %q", opts.project, opts.manifestName)
		}
		clear(projects)
		projects[proj.Name] = proj
	}

	log.DebugContext(ctx, "Deploying to accounts: %q", maps.Keys(mani.Accounts))
	log.DebugContext(ctx, "Deploying projects: %q", maps.Keys(projects))

	resources, err := loader.LoadResources(fs, opts.workingDir, projects)
	if err != nil {
		return fmt.Errorf("failed to load all account management resources: %w", err)
	}

	if opts.dryRun {
		log.InfoContext(ctx, "Successfully validated account management resources")
		return nil
	}

	accountClients, err := dynatrace.CreateAccountClients(ctx, mani.Accounts)
	if err != nil {
		return fmt.Errorf("failed to create account clients: %w", err)
	}

	maxConcurrentDeploys := environment.GetEnvValueInt(environment.ConcurrentRequestsEnvKey)

	for accInfo, accClient := range accountClients {
		logger := log.With(slog.Any("account", accInfo.Name))
		accountDeployer := deployer.NewAccountDeployer(deployer.NewClient(accInfo, accClient), deployer.WithMaxConcurrentDeploys(maxConcurrentDeploys))
		logger.InfoContext(ctx, "Deploying configuration for account '%s' (%s)", accInfo.Name, accInfo.AccountUUID)
		logger.InfoContext(ctx, "Number of users to deploy: %d", len(resources.Users))
		logger.InfoContext(ctx, "Number of service users to deploy: %d", len(resources.ServiceUsers))
		logger.InfoContext(ctx, "Number of groups to deploy: %d", len(resources.Groups))
		logger.InfoContext(ctx, "Number of policies to deploy: %d", len(resources.Policies))

		if err = accountDeployer.Deploy(ctx, resources); err != nil {
			return err
		}
	}

	return nil
}
