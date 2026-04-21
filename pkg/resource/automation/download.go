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
	"log/slog"
	"sync"
	"time"

	"golang.org/x/exp/maps"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	templateEscaper "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DownloadSource interface {
	List(context.Context, automation.ResourceType) (api.PagedListResponse, error)
	Get(ctx context.Context, resourceType automation.ResourceType, id string) (api.Response, error)
}

type DownloadAPI struct {
	automationSource DownloadSource
}

func NewDownloadAPI(automationSource DownloadSource) *DownloadAPI {
	return &DownloadAPI{automationSource}
}

// Download downloads all automation resources for a given project
func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.InfoContext(ctx, "Downloading automation resources")
	configsPerType := make(project.ConfigsPerType)
	for _, at := range maps.Keys(automationTypesToResources) {
		lg := log.With(log.TypeAttr(at.Resource))

		resource, ok := automationTypesToResources[at]
		if !ok {
			lg.WarnContext(ctx, "No resource mapping for automation type %s found", at.Resource)
			continue
		}
		response, err := func() (api.PagedListResponse, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			return a.automationSource.List(ctx, resource)
		}()

		if err != nil {
			lg.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to fetch all objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		objects, err := automationutils.DecodeListResponse(response)
		if err != nil {
			lg.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to decode API response objects for automation resource %s: %v", at.Resource, err)
			continue
		}

		if len(objects) == 0 {
			// Info on purpose. Most types have a lot of objects, so skipping printing 'not found' in the default case makes sense. Here it's kept on purpose, we have only 3 types.
			lg.With(slog.Any("configsDownloaded", len(objects))).InfoContext(ctx, "Did not find any %s to download", string(at.Resource))

			continue
		}
		lg.With(slog.Any("configsDownloaded", len(objects))).InfoContext(ctx, "Downloaded %d objects for %s", len(objects), string(at.Resource))

		if resource == automation.Workflows {
			// for workflows, we need to fetch the full object with Get, as the List endpoint only returns a subset of the properties, and we want to have them all for the template
			objects = a.fetchFullObjects(ctx, lg, objects, projectName)
		}

		var configs []config.Config
		for _, obj := range objects {

			configId := obj.ID

			if escaped, err := escapeJinjaTemplates(obj.Data); err != nil {
				lg.With(log.CoordinateAttr(coordinate.Coordinate{Project: projectName, Type: string(at.Resource), ConfigId: configId}), log.ErrorAttr(err)).WarnContext(ctx, "Failed to escape automation templating expressions for config %v (%s) - template needs manual adaptation: %v", configId, at.Resource, err)
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

// fetchFullObjects fetches the full representation of each object by calling Get for every entry in objects.
// This is necessary because some List endpoints (e.g. Workflows) only return a subset of each object's properties.
func (a DownloadAPI) fetchFullObjects(ctx context.Context, lg *log.Slogger, objects []automationutils.Response, projectName string) []automationutils.Response {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	full := make([]automationutils.Response, 0, len(objects))

	for _, obj := range objects {
		wg.Go(func() {
			cfgLg := lg.With(log.CoordinateAttr(coordinate.Coordinate{Project: projectName, Type: string(config.Workflow), ConfigId: obj.ID}))
			resp, err := a.automationSource.Get(ctx, automation.Workflows, obj.ID)
			if err != nil {
				cfgLg.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to fetch full object for config %v (%s): %v", obj.ID, string(config.Workflow), err)
				return
			}
			decoded, err := automationutils.DecodeResponse(resp)
			if err != nil {
				cfgLg.With(log.ErrorAttr(err)).ErrorContext(ctx, "Failed to decode full object for config %v (%s): %v", obj.ID, string(config.Workflow), err)
				return
			}
			mutex.Lock()
			defer mutex.Unlock()
			full = append(full, decoded)
		})
	}
	wg.Wait()

	return full
}

func escapeJinjaTemplates(src []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, src, "", "\t")
	return templateEscaper.UseGoTemplatesForDoubleCurlyBraces(prettyJSON.Bytes()), err
}

func createTemplateFromRawJSON(obj automationutils.Response, configType, projectName string) (t template.Template, extractedName *string) {
	configId := obj.ID

	var data map[string]any
	err := json.Unmarshal(obj.Data, &data)
	if err != nil {
		log.With(log.CoordinateAttr(coordinate.Coordinate{Project: projectName, Type: configType, ConfigId: configId}), log.ErrorAttr(err)).Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
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
		log.With(log.CoordinateAttr(coordinate.Coordinate{Project: projectName, Type: configType, ConfigId: configId}), log.ErrorAttr(err)).Warn("Failed to sanitize downloaded JSON for config %v (%s) - template may need manual cleanup: %v", configId, configType, err)
		content = obj.Data
	}
	content = jsonutils.MarshalIndent(content)

	t = template.NewInMemoryTemplate(configId, string(content))
	return t, extractedName
}
