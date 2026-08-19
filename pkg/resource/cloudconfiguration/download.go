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

package cloudconfiguration

import (
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DownloadSource interface {
	List(ctx context.Context) (api.ListResponse, error)
}

type DownloadAPI struct {
	source DownloadSource
}

func NewDownloadAPI(source DownloadSource) *DownloadAPI {
	return &DownloadAPI{source}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.InfoContext(ctx, "Downloading Cloud Configurations")

	items, err := a.source.List(ctx)
	if err != nil {
		log.With(log.TypeAttr(config.CloudConfigurationID), log.ErrorAttr(err)).ErrorContext(ctx, "Failed to fetch the list of existing %s configs: %v", config.CloudConfigurationID, err)
		return nil, nil
	}

	var configs []config.Config
	for _, raw := range items.Objects {
		c, err := toConfig(projectName, raw)
		if err != nil {
			log.With(log.TypeAttr(config.CloudConfigurationID), log.ErrorAttr(err)).ErrorContext(ctx, "Failed to convert %s: %v", config.CloudConfigurationID, err)
			continue
		}
		configs = append(configs, c)
	}

	return project.ConfigsPerType{string(config.CloudConfigurationID): configs}, nil
}

func toConfig(projectName string, data []byte) (config.Config, error) {
	obj, err := templatetools.NewJSONObject(data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := obj.Get("id").(string)
	if !ok || id == "" {
		return config.Config{}, fmt.Errorf("API payload is missing 'id'")
	}

	obj.Delete("id", "tenantUuid", "createdAt", "updatedAt")

	jsonRaw, err := obj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.CloudConfigurationID),
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.CloudConfiguration{},
		Parameters:     make(config.Parameters),
	}, nil
}
