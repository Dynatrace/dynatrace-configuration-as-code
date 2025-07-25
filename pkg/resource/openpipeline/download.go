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

package openpipeline

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
	GetAll(context.Context) ([]api.Response, error)
}

type DownloadAPI struct {
	openPipelineSource DownloadSource
}

func NewDownloadAPI(source DownloadSource) *DownloadAPI {
	return &DownloadAPI{source}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.InfoContext(ctx, "Downloading openpipelines")
	result := project.ConfigsPerType{string(config.OpenPipelineTypeID): nil}
	all, err := a.openPipelineSource.GetAll(ctx)
	if err != nil {
		log.With(log.TypeAttr(config.OpenPipelineTypeID), log.ErrorAttr(err)).ErrorContext(ctx, "Failed to get all configs of type '%s': %v", config.OpenPipelineTypeID, err)
		return result, nil
	}

	var configs []config.Config
	for _, response := range all {
		c, err := createConfig(projectName, response)
		if err != nil {
			log.With(log.TypeAttr(config.OpenPipelineTypeID), log.ErrorAttr(err)).ErrorContext(ctx, "Failed to convert config of type '%s': %v", config.OpenPipelineTypeID, err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.OpenPipelineTypeID)] = configs

	return result, nil
}

func createConfig(projectName string, response api.Response) (config.Config, error) {
	jsonObj, err := templatetools.NewJSONObject(response.Data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get("id").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("failed to extract id as string from payload")
	}

	// delete fields that prevent a re-upload of the configuration
	jsonObj.Delete("version", "updateToken")

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.OpenPipelineTypeID),
			ConfigId: id,
		},
		Type: config.OpenPipelineType{
			Kind: id,
		},
		Parameters: make(config.Parameters),
	}, nil
}
