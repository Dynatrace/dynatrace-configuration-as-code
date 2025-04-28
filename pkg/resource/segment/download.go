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

package segment

import (
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/segments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type Source interface {
	GetAll(ctx context.Context) ([]segments.Response, error)
}

type Api struct {
	segmentSource Source
}

func NewApi(segmentSource Source) *Api {
	return &Api{segmentSource}
}

func (a Api) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	result := project.ConfigsPerType{}

	downloadedConfigs, err := a.segmentSource.GetAll(ctx)
	if err != nil {
		log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to fetch the list of existing segments: %v", err)
		return nil, nil
	}

	var configs []config.Config
	for _, downloadedConfig := range downloadedConfigs {
		c, err := createConfig(projectName, downloadedConfig)
		if err != nil {
			log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to convert segment: %v", err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.SegmentID)] = configs

	return result, nil
}

func createConfig(projectName string, response segments.Response) (config.Config, error) {
	jsonObj, err := templatetools.NewJSONObject(response.Data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get("uid").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("API payload is missing 'uid'")
	}

	// delete fields that prevent a re-upload of the configuration
	jsonObj.Delete("uid", "version", "externalId")

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.SegmentID),
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.Segment{},
		Parameters:     make(config.Parameters),
	}, nil
}
