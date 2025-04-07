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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/report"
)

// preloadCaches fills the caches of the specified clients for the config types used in the given projects.
func preloadCaches(ctx context.Context, environment project.Environment, clientSet *client.ClientSet) {
	var wg sync.WaitGroup
	for _, c := range environment.PreloadConfigTypes() {
		wg.Add(1)
		go func(configType config.Type) {
			defer wg.Done()

			switch t := configType.(type) {
			case config.SettingsType:
				if clientSet.SettingsClient != nil {
					preloadSettingsValuesForSchemaId(ctx, clientSet.SettingsClient, t.SchemaId)
				}

			case config.ClassicApiType:
				if clientSet.ConfigClient != nil {
					preloadValuesForApi(ctx, clientSet.ConfigClient, t.Api)
				}
			}

		}(c)
	}
	wg.Wait()
}

func clearCaches(clientSet *client.ClientSet) {
	if clientSet.SettingsClient != nil {
		clientSet.SettingsClient.ClearCache()
	}

	if clientSet.ConfigClient != nil {
		clientSet.ConfigClient.ClearCache()
	}
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
