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
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/automation/internal"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
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

		objects := response.All()

		configs := convertAllObjects(projectName, string(at.Resource), objects)

		log.WithFields(field.Type(string(at.Resource)), field.F("configsDownloaded", len(configs))).
			Info("Downloaded %d objects for automation resource %s", len(configs), string(at.Resource))

		configsPerType[string(at.Resource)] = configs
	}
	return configsPerType, nil
}

func convertAllObjects(projectName, resource string, objects [][]byte) []config.Config {
	var result []config.Config
	for _, o := range objects {
		c, err := convertObject(o)
		if err != nil {
			log.WithFields(field.Type(resource), field.Error(err)).
				Error("Failed to decode API response for %q resource: %v", resource, err)
			continue
		}
		c.Coordinate.Project = projectName
		c.Coordinate.Type = resource
		c.Type = config.AutomationType{Resource: config.AutomationResource(resource)}

		result = append(result, c)
	}
	return result
}

func convertObject(o []byte) (config.Config, error) {

	if escaped, err := escapeJinjaTemplates(o); err != nil {
		return config.Config{}, fmt.Errorf("failed to escape automation templating expressions - template needs manual adaptation: %w", err)
	} else {
		o = escaped
	}

	r, err := templatetools.NewJSONObject(o)
	if err != nil {
		return config.Config{}, err
	}

	configID := r.Get("id").(string)

	var configName string
	if r.Get("title") != nil {
		configName = r.Get("title").(string)
	} else {
		configName = configID
	}

	params := map[string]parameter.Parameter{}
	if p := r.ParameterizeAttributeWith("title", "name"); p != nil {
		params[config.NameParameter] = p
	}

	r.Delete("id")
	r.Delete("modificationInfo")
	r.Delete("lastExecution")

	t, err := r.ToJSON()
	if err != nil {
		return config.Config{}, err
	}

	return config.Config{
		Template: template.NewDownloadTemplate(configID, configName, string(jsonutils.MarshalIndent(t))),
		Coordinate: coordinate.Coordinate{
			ConfigId: configID,
		},
		Parameters:     params,
		OriginObjectId: configID,
	}, nil
}

func escapeJinjaTemplates(src []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, src, "", "\t")
	return internal.EscapeJinjaTemplates(prettyJSON.Bytes()), err
}
