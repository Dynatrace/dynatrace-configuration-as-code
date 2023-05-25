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
	"bytes"
	"encoding/json"
	"fmt"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	client "github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/automation"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/download/automation/internal"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/pkg/project/v2"
	"golang.org/x/exp/maps"
)

var automationTypesToResources = map[config.AutomationType]client.ResourceType{
	config.AutomationType{Resource: config.Workflow}:         client.Workflows,
	config.AutomationType{Resource: config.BusinessCalendar}: client.BusinessCalendars,
	config.AutomationType{Resource: config.SchedulingRule}:   client.SchedulingRules,
}

// Downloader can be used to download automation resources/configs
type Downloader struct {
	client *client.Client
}

// NewDownloader creates a new [Downloader] for automation resources/configs
func NewDownloader(client *client.Client) *Downloader {
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

		log.Info("Downloaded %d objects for automation resource %s", len(*response), string(at.Resource))
		if len(*response) == 0 {
			continue
		}

		var configs []config.Config
		for _, obj := range *response {

			configId := obj.Id

			if escaped, err := escapeJinjaTemplates(obj.Data); err != nil {
				log.Warn("Failed to escape automation templating expressions for config %v (%s) - template needs manual adaptation: %v", configId, at.Resource, err)
			} else {
				obj.Data = escaped
			}

			t, configName := createTemplateFromRawJSON(obj, string(at.Resource))

			c := config.Config{
				Template: t,
				Coordinate: coordinate.Coordinate{
					Project:  projectName,
					Type:     string(at.Resource),
					ConfigId: configId,
				},
				Type: config.AutomationType{
					Resource: at.Resource,
				},
				Parameters: map[string]parameter.Parameter{
					config.NameParameter: &value.ValueParameter{Value: configName},
				},
				OriginObjectId: obj.Id,
			}
			configs = append(configs, c)
		}
		configsPerType[string(at.Resource)] = configs
	}
	return configsPerType, nil
}

func escapeJinjaTemplates(src []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, src, "", "\t")
	return internal.EscapeJinjaTemplates(prettyJSON.Bytes()), err
}

type NoopAutomationDownloader struct {
}

func createTemplateFromRawJSON(obj client.Response, configType string) (t template.Template, extractedName string) {
	configId := obj.Id

	var data map[string]interface{}
	err := json.Unmarshal(obj.Data, &data)
	if err != nil {
		log.Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
		return template.NewDownloadTemplate(configId, configId, string(obj.Data)), configId
	}

	// remove properties not necessary for upload
	delete(data, "id")
	delete(data, "modificationInfo")
	delete(data, "lastExecution")

	// extract 'title' as name
	configName := configId
	if title, ok := data["title"]; ok {
		configName = fmt.Sprintf("%v", title)
		data["title"] = "{{.name}}"
	}

	var content []byte
	if modifiedJson, err := json.Marshal(data); err == nil {
		content = modifiedJson
	} else {
		log.Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
		content = obj.Data
	}
	content = jsonutils.MarshalIndent(content)

	t = template.NewDownloadTemplate(configId, configName, string(content))
	return t, configName
}

func (d NoopAutomationDownloader) Download(_ string, _ ...config.AutomationType) (v2.ConfigsPerType, error) {
	return nil, nil
}
