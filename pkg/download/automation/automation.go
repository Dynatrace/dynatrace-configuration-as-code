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

package automation

import (
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/maps"
	automationClient "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
)

var automationTypesToResources = map[config.AutomationType]automationClient.ResourceType{
	config.AutomationType{Resource: config.Workflow}:         automationClient.Workflows,
	config.AutomationType{Resource: config.BusinessCalendar}: automationClient.BusinessCalendars,
	config.AutomationType{Resource: config.SchedulingRule}:   automationClient.SchedulingRules,
}

// Downloader can be used to download automation resources/configs
type Downloader struct {
	client *automationClient.Client
}

// NewDownloader creates a new [Downloader] for automation resources/configs
func NewDownloader(client *automationClient.Client) *Downloader {
	return &Downloader{
		client: client,
	}
}

// Download downloads all automation resources for a given project
// If automationTypes is given it will just download those types of automation resources
func (d *Downloader) Download(projectName string, automationTypes ...config.AutomationType) (v2.ConfigsPerType, error) {
	if len(automationTypes) == 0 {
		automationTypes = maps.Keys(automationTypesToResources)
	}

	configsPerType := make(v2.ConfigsPerType)
	for _, at := range automationTypes {
		resource, ok := automationTypesToResources[at]
		if !ok {
			log.Warn("No resource mapping for automation type %s found", at.Resource)
			continue
		}
		response, err := d.client.List(resource)
		if err != nil {
			log.Error("Failed to fetch all objects for automation resource %s: %v", err)
			continue
		}

		log.Info("Downloaded %d objects for automation resource %s", len(response.Results), string(at.Resource))
		if len(response.Results) == 0 {
			continue
		}

		var configs []config.Config
		for _, obj := range response.Results {
			configId := obj.Id
			content, _ := jsonutils.MarshalIndent(obj.Data)
			c := config.Config{
				Template: template.NewDownloadTemplate(configId, configId, string(content)),
				Coordinate: coordinate.Coordinate{
					Project:  projectName,
					Type:     string(at.Resource),
					ConfigId: configId,
				},
				Type: config.AutomationType{
					Resource: at.Resource,
				},
				Parameters: map[string]parameter.Parameter{
					config.NameParameter: &value.ValueParameter{Value: configId},
				},
				OriginObjectId: obj.Id,
			}
			configs = append(configs, c)
		}
		configsPerType[string(at.Resource)] = configs
	}
	return configsPerType, nil
}

type NoopAutomationDownloader struct {
}

func (d NoopAutomationDownloader) Download(_ string, _ ...config.AutomationType) (v2.ConfigsPerType, error) {
	return nil, nil
}
