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

package grailfiltersegment

import (
	"context"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

func Download(client client.GrailFilterSegmentClient, projectName string) (project.ConfigsPerType, error) {
	result := project.ConfigsPerType{}

	dtos, err := client.GetAll(context.TODO())
	if err != nil {
		log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to fetch the list of existing filter-segments: %v", err)
		return nil, nil
	}

	var configs []config.Config
	for _, dto := range dtos {
		c, err := createConfig(projectName, dto)
		if err != nil {
			log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to convert config of type '%s': %v", config.SegmentID, err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.SegmentID)] = configs

	return result, nil
}

func createConfig(projectName string, response openpipeline.Response) (config.Config, error) {
	jsonObj, err := templatetools.NewJSONObject(response.Data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get("uid").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("failed to extract id as string from payload")
	}

	jsonObj.Delete("uid")
	jsonObj.Delete("version")
	jsonObj.Delete("externalId")

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.SettingsTypeID),
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.Segment{},
		Parameters:     make(config.Parameters),
	}, nil
}
