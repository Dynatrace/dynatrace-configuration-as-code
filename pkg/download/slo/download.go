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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/configcreation"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type requiredSloProps struct {
	ID string `json:"id"`
}

type DownloadSloClient interface {
	List(ctx context.Context) (api.PagedListResponse, error)
}

func Download(client DownloadSloClient, projectName string) (project.ConfigsPerType, error) {
	result := project.ConfigsPerType{}
	downloadedConfigs, err := client.List(context.TODO())
	if err != nil {
		log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to fetch the list of existing SLOs: %v", err)
		return nil, nil
	}

	allConfigs := downloadedConfigs.All()
	configs := make([]config.Config, 0, len(allConfigs))
	for _, downloadedConfig := range allConfigs {
		var requiredProps requiredSloProps
		preparedConfig, err := configcreation.PrepareConfig(downloadedConfig, &requiredProps, []string{"id", "version", "externalId"}, "")
		if err != nil {
			log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to convert SLO: %v", err)
			continue
		}
		if requiredProps.ID == "" {
			err = fmt.Errorf("API payload is missing 'id'")
			log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to convert SLO: %v", err)
			continue
		}
		c := config.Config{
			Template: template.NewInMemoryTemplate(requiredProps.ID, preparedConfig.JSONString),
			Coordinate: coordinate.Coordinate{
				Project:  projectName,
				Type:     string(config.ServiceLevelObjectiveID),
				ConfigId: requiredProps.ID,
			},
			OriginObjectId: requiredProps.ID,
			Type:           config.ServiceLevelObjective{},
			Parameters:     make(config.Parameters),
		}
		configs = append(configs, c)
	}
	result[string(config.ServiceLevelObjectiveID)] = configs

	return result, nil
}
