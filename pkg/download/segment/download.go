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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/config_creation"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type requiredSegmentProps struct {
	UId string `json:"uid"`
}

type DownloadSegmentClient interface {
	GetAll(ctx context.Context) ([]segments.Response, error)
}

func Download(client DownloadSegmentClient, projectName string) (project.ConfigsPerType, error) {
	result := project.ConfigsPerType{}

	downloadedConfigs, err := client.GetAll(context.TODO())
	if err != nil {
		log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to fetch the list of existing segments: %v", err)
		return nil, nil
	}

	var configs []config.Config
	for _, downloadedConfig := range downloadedConfigs {
		var requiredProps requiredSegmentProps
		preparedConfig, err := config_creation.PrepareConfig(downloadedConfig.Data, &requiredProps, []string{"uid", "version", "externalId"}, "")
		if err != nil {
			log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to convert segment: %v", err)
			continue
		}
		if requiredProps.UId == "" {
			err = fmt.Errorf("API payload is missing 'uid'")
			log.WithFields(field.Type(config.SegmentID), field.Error(err)).Error("Failed to convert SLO: %v", err)
			continue
		}
		c := config.Config{
			Template: template.NewInMemoryTemplate(requiredProps.UId, preparedConfig.JSONString),
			Coordinate: coordinate.Coordinate{
				Project:  projectName,
				Type:     string(config.SegmentID),
				ConfigId: requiredProps.UId,
			},
			OriginObjectId: requiredProps.UId,
			Type:           config.Segment{},
			Parameters:     make(config.Parameters),
		}
		configs = append(configs, c)
	}
	result[string(config.SegmentID)] = configs

	return result, nil
}
