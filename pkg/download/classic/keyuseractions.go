/*
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

package classic

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
)

func (d *Downloader) downloadKeyUserActions(projectName string) []downloadedConfig {

	var configs []downloadedConfig

	keyUserActionApi := api.NewAPIs()["key-user-actions-mobile"]
	appMobile, err := d.client.ListConfigs(context.TODO(), api.NewAPIs()["application-mobile"])
	if err != nil {
		return configs
	}
	// grab all mobile applications
	for _, a := range appMobile {
		values, err := d.client.ListConfigs(context.TODO(), keyUserActionApi.Resolve(a.Id))
		if err != nil {
			return configs
		}
		values = d.filterConfigsToSkip(keyUserActionApi, values)
		// grab all key user actions for each application
		for _, v := range values {
			mappedJson := map[string]any{"name": v.Id}
			cfg, err := d.createConfigForDownloadedJson(mappedJson, keyUserActionApi, v, projectName)
			if err != nil {
				return configs
			}
			// set scope parameter because this config is referencing another config (application-mobile) via its url path
			cfg.Parameters[config.ScopeParameter] = reference.New(projectName, "application-mobile", a.Id, "id")
			configs = append(configs, downloadedConfig{Config: cfg, value: v})
		}
	}

	return configs
}
