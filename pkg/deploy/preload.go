/**
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"context"
	"fmt"
	"sync"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/dynatrace"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

type preloadConfigTypeEntry struct {
	configType config.Type
	clientset  *client.ClientSet
}

// preloadCaches fills the caches of the specified clients for the config types used in the given projects.
func preloadCaches(ctx context.Context, projects []project.Project, environmentClients dynatrace.EnvironmentClients) {
	var wg sync.WaitGroup
	for _, p := range gatherPreloadConfigTypeEntries(projects, environmentClients) {
		wg.Add(1)
		go func(p preloadConfigTypeEntry) {
			defer wg.Done()

			switch t := p.configType.(type) {
			case config.SettingsType:
				preloadSettingsValuesForSchemaId(ctx, p.clientset.SettingsClient, t.SchemaId)

			case config.ClassicApiType:
				preloadValuesForApi(ctx, p.clientset.ConfigClient, t.Api)
			}

		}(p)
	}
	wg.Wait()
}

func preloadSettingsValuesForSchemaId(ctx context.Context, client client.SettingsClient, schemaId string) {
	if err := client.Cache(ctx, schemaId); err != nil {
		message := fmt.Sprintf("Could not cache settings values for schema %s: %s", schemaId, err)
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateWarn, nil, message, nil)
		log.Warn(message)
		return
	}
	message := fmt.Sprintf("Cached settings values for schema %s", schemaId)
	log.Debug(message)
	report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateSuccess, nil, message, nil)
}

func preloadValuesForApi(ctx context.Context, client client.ConfigClient, theApi string) {
	a, ok := api.NewAPIs()[theApi]
	if !ok {
		return
	}
	if a.HasParent() {
		return
	}
	err := client.Cache(ctx, a)
	if err != nil {
		message := fmt.Sprintf("Could not cache values for API %s: %s", theApi, err)
		report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateWarn, nil, message, nil)
		log.Warn(message)
		return
	}
	message := fmt.Sprintf("Cached values for API %s", theApi)
	report.GetReporterFromContextOrDiscard(ctx).ReportLoading(report.StateSuccess, nil, message, nil)
	log.Debug(message)
}

// gatherPreloadConfigTypeEntries scans the projects to determine which config types should be cached by which clients.
func gatherPreloadConfigTypeEntries(projects []project.Project, environmentClients dynatrace.EnvironmentClients) []preloadConfigTypeEntry {
	preloads := []preloadConfigTypeEntry{}
	for environmentInfo, environmentClientSet := range environmentClients {
		seenConfigTypes := map[string]struct{}{}

		for _, project := range projects {
			project.ForEveryConfigInEnvironmentDo(environmentInfo.Name, func(c config.Config) {
				// If the config shall be skipped there is no point in caching it
				if c.Skip {
					return
				}
				if _, ok := seenConfigTypes[c.Coordinate.Type]; ok {
					return
				}
				seenConfigTypes[c.Coordinate.Type] = struct{}{}

				switch t := c.Type.(type) {
				case config.ClassicApiType:
					if environmentClientSet.ConfigClient != nil {
						preloads = append(preloads, preloadConfigTypeEntry{configType: t, clientset: environmentClientSet})
					}

				case config.SettingsType:
					if environmentClientSet.SettingsClient != nil {
						preloads = append(preloads, preloadConfigTypeEntry{configType: t, clientset: environmentClientSet})
					}
				}
			})
		}
	}
	return preloads
}
