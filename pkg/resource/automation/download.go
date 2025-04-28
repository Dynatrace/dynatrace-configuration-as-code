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
	"time"

	"golang.org/x/exp/maps"

	automationAPI "github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	templateEscaper "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

var automationTypesToResources = map[config.AutomationType]automationAPI.ResourceType{
	config.AutomationType{Resource: config.Workflow}:         automationAPI.Workflows,
	config.AutomationType{Resource: config.BusinessCalendar}: automationAPI.BusinessCalendars,
	config.AutomationType{Resource: config.SchedulingRule}:   automationAPI.SchedulingRules,
}

// Download downloads all automation resources for a given project
// If automationTypes is given it will just download those types of automation resources
func Download(ctx context.Context, cl client.AutomationClient, projectName string, automationTypes ...config.AutomationType) (project.ConfigsPerType, error) {
	if len(automationTypes) == 0 {
		automationTypes = maps.Keys(automationTypesToResources)
	}

	configsPerType := make(project.ConfigsPerType)
	for _, at := range automationTypes {
		lg := log.WithFields(field.Type(at.Resource))

		resource, ok := automationTypesToResources[at]
		if !ok {
			lg.Warn("No resource mapping for automation type %s found", at.Resource)
			continue
		}
		response, err := func() (automation.ListResponse, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			return cl.List(ctx, resource)
		}()

		if err != nil {
			lg.WithFields(field.Error(err)).Error("Failed to fetch all objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		objects, err := automationutils.DecodeListResponse(response)
		if err != nil {
			lg.WithFields(field.Error(err)).Error("Failed to decode API response objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		if len(objects) == 0 {
			// Info on purpose. Most types have a lot of objects, so skipping printing 'not found' in the default case makes sense. Here it's kept on purpose, we have only 3 types.
			lg.WithFields(field.F("configsDownloaded", len(objects))).Info("Did not find any %s to download", string(at.Resource))

			continue
		}
		lg.WithFields(field.F("configsDownloaded", len(objects))).Info("Downloaded %d objects for %s", len(objects), string(at.Resource))

		var configs []config.Config
		for _, obj := range objects {

			configId := obj.ID

			if escaped, err := escapeJinjaTemplates(obj.Data); err != nil {
				lg.WithFields(field.Coordinate(coordinate.Coordinate{Project: projectName, Type: string(at.Resource), ConfigId: configId}), field.Error(err)).Warn("Failed to escape automation templating expressions for config %v (%s) - template needs manual adaptation: %v", configId, at.Resource, err)
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
	return templateEscaper.UseGoTemplatesForDoubleCurlyBraces(prettyJSON.Bytes()), err
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
