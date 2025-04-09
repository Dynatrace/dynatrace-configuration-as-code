/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package slo

import (
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DownloadSloClient interface {
	List(ctx context.Context) (api.PagedListResponse, error)
}

func Download(ctx context.Context, client DownloadSloClient, projectName string) (project.ConfigsPerType, error) {
	result := project.ConfigsPerType{}
	downloadedConfigs, err := client.List(ctx)
	if err != nil {
		log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to fetch the list of existing %s configs: %v", config.ServiceLevelObjectiveID, err)
		// error is ignored
		return nil, nil
	}

	var configs []config.Config
	for _, downloadedConfig := range downloadedConfigs.All() {
		c, err := createConfig(projectName, downloadedConfig)
		if err != nil {
			log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to convert %s: %v", config.ServiceLevelObjectiveID, err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.ServiceLevelObjectiveID)] = configs

	return result, nil
}

func createConfig(projectName string, data []byte) (config.Config, error) {
	jsonObj, err := templatetools.NewJSONObject(data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get("id").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("API payload is missing 'id'")
	}

	// delete fields that prevent a re-upload of the configuration
	jsonObj.Delete("id", "version", "externalId")

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.ServiceLevelObjectiveID),
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.ServiceLevelObjective{},
		Parameters:     make(config.Parameters),
	}, nil
}
