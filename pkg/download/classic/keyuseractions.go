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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/reference"
	"github.com/mitchellh/mapstructure"
)

func (d *Downloader) downloadKeyUserActions(theApi api.API, projectName string, applicationType string) []downloadedConfig {
	var configs []downloadedConfig
	apps, err := d.client.ListConfigs(context.TODO(), api.NewAPIs()[applicationType])
	if err != nil {
		return configs
	}
	for _, a := range apps {
		kuas, err := d.downloadAndUnmarshalConfig(theApi.Resolve(a.Id), dtclient.Value{})
		if err != nil {
			return configs
		}
		var keyUserActions dtclient.KeyUserActionsMobileResponse
		mapstructure.Decode(kuas, &keyUserActions)

		var arr []map[string]any
		mapstructure.Decode(kuas[theApi.PropertyNameOfGetAllResponse], &arr)
		for _, content := range arr {
			value := dtclient.Value{Id: content["name"].(string), Name: content["name"].(string)}
			cfg, err := d.createConfigForDownloadedJson(content, theApi, value, projectName)
			if err != nil {
				return configs
			}
			cfg.Parameters[config.ScopeParameter] = reference.New(projectName, applicationType, a.Id, "id")
			configs = append(configs, downloadedConfig{Config: cfg, value: value})

		}
	}

	return configs
}
