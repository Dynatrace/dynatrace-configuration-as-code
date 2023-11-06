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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/deployer"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/persistence/account"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

type deployOpts struct {
	manifestName string
	accountName  string
	project      string
	dryRun       bool
}

func deployCommand(fs afero.Fs) *cobra.Command {
	opts := deployOpts{}

	command := &cobra.Command{
		Use:               "deploy <manifest.yaml> [flags]",
		Short:             "Deploy account management resources",
		Example:           "monaco account deploy manifest.yaml [--account <account-name-in-manifest>] [--project <project-defined-in-manifest>]",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completion.SingleArgumentManifestFileCompletion,
		PreRun:            cmdutils.SilenceUsageCommand(),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestName := args[0]

			if !files.IsYamlFileExtension(manifestName) {
				return fmt.Errorf("expected a .yaml file, but got %s", manifestName)
			}

			opts.manifestName = manifestName

			return deploy(fs, opts)
		},
	}

	command.Flags().StringVarP(&opts.accountName, "account", "a", "", "Account name defined in the manifest to deploy to.")
	command.Flags().StringVarP(&opts.project, "project", "p", "", "Project name defined in the manifest")
	command.Flags().BoolVarP(&opts.dryRun, "dry-run", "d", false, "Validate the structure of your manifest, projects and configurations. Dry-run will resolve all configuration parameters but cannot verify if the content will be accepted by the Dynatrace APIs.")

	return command
}

func deploy(fs afero.Fs, opts deployOpts) error {

	mani, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: opts.manifestName,
	})
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return errors.New("error while loading manifest")
	}

	// filter account
	accs := mani.Accounts
	if opts.accountName != "" {
		if acc, f := accs[opts.accountName]; !f {
			return fmt.Errorf("required account %q was not found in manifest %q", opts.accountName, opts.manifestName)
		} else {
			clear(accs)
			accs[acc.Name] = acc
		}
	}

	// filter project
	projs := mani.Projects
	if opts.project != "" {
		if proj, f := projs[opts.project]; !f {
			return fmt.Errorf("required project %q was not found in manifest %q", opts.accountName, opts.manifestName)
		} else {
			clear(projs)
			projs[proj.Name] = proj
		}
	}

	log.Debug("Deploying to accounts: %q", maps.Keys(accs))
	log.Debug("Deploying projects: %q", maps.Keys(projs))

	resources, errs := loadAccountManagementResources(fs, projs)
	if len(errs) > 0 {
		errutils.PrintErrors(errs)
		return errors.New("failed to load all account management resources")
	}

	err := deployer.Deploy(accs, resources, deployer.Options{DryRun: opts.dryRun})
	return err
}

func loadAccountManagementResources(fs afero.Fs, projs manifest.ProjectDefinitionByProjectID) (map[string]*account.AMResources, []error) {
	resources := make(map[string]*account.AMResources, len(projs))
	var errs []error

	// load project content
	for _, p := range projs {
		if a, err := account.Load(fs, p.Path); err != nil {
			errs = append(errs, err)
		} else if err := account.Validate(a); err != nil {
			errs = append(errs, err)
		} else {
			resources[p.Name] = a
		}
	}

	return resources, errs
}
