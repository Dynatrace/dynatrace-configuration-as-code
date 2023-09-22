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
	"context"
	"encoding/json"
	"fmt"
	client "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation/internal"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
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
			log.WithFields(field.Type(string(at.Resource))).Warn("No resource mapping for automation type %s found", at.Resource)
			continue
		}
		response, err := d.client.List(context.TODO(), resource)
		if err != nil {
			log.WithFields(field.Type(string(at.Resource)), field.Error(err)).Error("Failed to fetch all objects for automation resource %s: %v", at.Resource, err)
			continue
		}
		if err, isAPIErr := response.AsAPIError(); isAPIErr {
			log.WithFields(field.Type(string(at.Resource)), field.Error(err)).Error("Failed to fetch all objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		objects, err := automationutils.DecodeListResponse(response)
		if err != nil {
			log.WithFields(field.Type(string(at.Resource)), field.Error(err)).Error("Failed to decode API response objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		log.WithFields(field.Type(string(at.Resource)), field.F("configsDownloaded", len(objects))).Info("Downloaded %d objects for automation resource %s", len(objects), string(at.Resource))
		if len(objects) == 0 {
			continue
		}

		var configs []config.Config
		for _, obj := range objects {

			configId := obj.ID

			if escaped, err := escapeJinjaTemplates(obj.Data); err != nil {
				log.WithFields(field.Coordinate(coordinate.Coordinate{Project: projectName, Type: string(at.Resource), ConfigId: configId}), field.Error(err)).Warn("Failed to escape automation templating expressions for config %v (%s) - template needs manual adaptation: %v", configId, at.Resource, err)
			} else {
				obj.Data = escaped
			}

			t, extractedName := createTemplateFromRawJSON(obj, string(at.Resource), projectName)

			params := map[string]parameter.Parameter{}
			if extractedName != nil {
				params[config.NameParameter] = &value.ValueParameter{Value: extractedName}
			}

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
				Parameters:     params,
				OriginObjectId: obj.ID,
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

func createTemplateFromRawJSON(obj automationutils.Response, configType, projectName string) (t template.Template, extractedName *string) {
	configId := obj.ID

	var data map[string]interface{}
	err := json.Unmarshal(obj.Data, &data)
	if err != nil {
		log.WithFields(field.Coordinate(coordinate.Coordinate{Project: projectName, Type: configType, ConfigId: configId}), field.Error(err)).Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
		return template.NewInMemoryTemplate(configId, string(obj.Data)), nil
	}

	// remove properties not necessary for upload
	delete(data, "id")
	delete(data, "modificationInfo")
	delete(data, "lastExecution")

	// extract 'title' as name
	configName := configId
	if title, ok := data["title"]; ok {
		configName = fmt.Sprintf("%v", title)
		extractedName = &configName

		data["title"] = "{{.name}}"
	}

	var content []byte
	if modifiedJson, err := json.Marshal(data); err == nil {
		content = modifiedJson
	} else {
		log.WithFields(field.Coordinate(coordinate.Coordinate{Project: projectName, Type: configType, ConfigId: configId}), field.Error(err)).Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
		content = obj.Data
	}
	content = jsonutils.MarshalIndent(content)

	t = template.NewInMemoryTemplate(configId, string(content))
	return t, extractedName
}

func (d NoopAutomationDownloader) Download(_ string, _ ...config.AutomationType) (v2.ConfigsPerType, error) {
	return nil, nil
}
