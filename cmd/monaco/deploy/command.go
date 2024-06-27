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

package deploy

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/cmdutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/completion"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/timeutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/events"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/subscribers"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func GetDeployCommand(fs afero.Fs) (deployCmd *cobra.Command) {
	var dryRun, continueOnError bool
	var manifestName string
	var environment, project, groups []string

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

			var eventbus events.EventSystem

			if featureflags.EventQueue().Enabled() {
				var err error
				eventbus, err = createDeployEventSystem(fs)
				if err != nil {
					return err
				}
			} else {
				eventbus = events.Discard()
			}

			eventbus.Start()
			defer eventbus.Terminate()

			return deployConfigs(fs, manifestName, groups, environment, project, continueOnError, dryRun, eventbus)

		},
	}

	deployCmd.Flags().StringSliceVarP(&environment, "environment", "e", []string{},
		"Specify one (or multiple) environment(s) to deploy to. "+
			"To set multiple environments either repeat this flag, or separate them using a comma (,). "+
			"This flag is mutually exclusive with '--group'.")
	deployCmd.Flags().StringSliceVarP(&groups, "group", "g", []string{},
		"Specify one (or multiple) environmentGroup(s) to deploy to. "+
			"To set multiple groups either repeat this flag, or separate them using a comma (,). "+
			"If this flag is specified, all environments within this group will be used for deployment. "+
			"This flag is mutually exclusive with '--environment'")
	deployCmd.Flags().StringSliceVarP(&project, "project", "p", make([]string, 0), "Project configuration to deploy (also deploys any dependent configurations)")
	deployCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Validate the structure of your manifest, projects and configurations. Dry-run will resolve all configuration parameters and render JSON templates, but can not validate the content of JSON payloads. After a successful dry-run, deployments may still fail with Dynatrace API errors if the content of JSONs is not valid.")
	deployCmd.Flags().BoolVarP(&continueOnError, "continue-on-error", "c", false, "Proceed deployment even if individual configuration deployments fail.")

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

func createDeployEventSystem(fs afero.Fs) (events.EventSystem, error) {
	ts := timeutils.TimeAnchor()

	fileSink, err := events.NewFileSink(fs, fmt.Sprintf("events-%s.txt", ts.Format("20060102150405")))
	if err != nil {
		return nil, err
	}

	ss, err := subscribers.NewDeploySubscriber(fileSink)
	if err != nil {
		return nil, fmt.Errorf("unable to create file collector for event queue: %w", err)
	}

	summaryFileSink, err := events.NewFileSink(fs, fmt.Sprintf("summary-%s.txt", ts.Format("20060102150405")))
	if err != nil {
		return nil, err
	}
	sc := &subscribers.SummarySubscriber{
		Sink: summaryFileSink,
	}

	eventbus, err := events.New(100, events.WithSubscribers(sc), events.WithSubscribers(ss))
	if err != nil {
		return nil, err
	}
	return eventbus, nil
}
