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
	"log/slog"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DownloadSource interface {
	GetAll(ctx context.Context) ([]api.Response, error)
}

type DownloadAPI struct {
	segmentSource DownloadSource
}

func NewDownloadAPI(segmentSource DownloadSource) *DownloadAPI {
	return &DownloadAPI{segmentSource}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	slog.InfoContext(ctx, "Downloading segments")
	result := project.ConfigsPerType{}

	lg := slog.With(log.TypeAttr(config.SegmentID))
	downloadedConfigs, err := a.segmentSource.GetAll(ctx)
	if err != nil {
		lg.ErrorContext(ctx, "Failed to fetch the list of existing segments: %v", err)
		return nil, nil
	}

	countReadyMade := 0
	var configs []config.Config
	for _, downloadedConfig := range downloadedConfigs {
		jsonConfig, err := unmarshalConfig(downloadedConfig)
		if err != nil {
			lg.ErrorContext(ctx, "Failed to convert segment: %v", err)
			continue
		}
		if isReadyMadeSegment(jsonConfig) {
			countReadyMade++
			continue
		}
		c, err := createConfig(projectName, jsonConfig)
		if err != nil {
			lg.ErrorContext(ctx, "Failed to convert segment: %v", err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.SegmentID)] = configs
	logDownloadResult(ctx, lg, len(downloadedConfigs), len(configs), countReadyMade)

	return result, nil
}

func logDownloadResult(ctx context.Context, lg *slog.Logger, totalDownloaded int, totalPersisted, readyMade int) {
	if totalDownloaded == 0 {
		lg.DebugContext(ctx, "Did not find any segments to download")
	} else {
		if readyMade > 0 {
			lg.InfoContext(ctx, "Downloaded segments. Skipped persisting ready-made segments", slog.Int("count", totalPersisted), slog.Int("skipCount", readyMade))
		} else {
			lg.InfoContext(ctx, "Downloaded segments", slog.Int("count", totalPersisted))
		}
	}
}

func unmarshalConfig(response api.Response) (templatetools.JSONObject, error) {
	jsonConfig, err := templatetools.NewJSONObject(response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return jsonConfig, nil
}

func isReadyMadeSegment(jsonConfig templatetools.JSONObject) bool {
	isReadyMade, ok := jsonConfig.Get("isReadyMade").(bool)
	// If the field is missing or not a bool, we assume it's not a ready-made segment
	return ok && isReadyMade
}

func createConfig(projectName string, jsonConfig templatetools.JSONObject) (config.Config, error) {
	id, ok := jsonConfig.Get("uid").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("API payload is missing 'uid'")
	}

	// delete fields that prevent a re-upload of the configuration
	jsonConfig.Delete("uid", "version", "externalId", "owner")

	jsonRaw, err := jsonConfig.ToJSON(true)
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
