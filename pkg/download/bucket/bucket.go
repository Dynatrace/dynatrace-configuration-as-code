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

package bucket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"strings"
)

type BucketClient interface {
	List(ctx context.Context) (buckets.ListResponse, error)
}

type Downloader struct {
	client BucketClient
}

func NewDownloader(client BucketClient) *Downloader {
	return &Downloader{
		client: client,
	}
}
func (d *Downloader) Download(projectName string, _ ...config.BucketType) (v2.ConfigsPerType, error) {
	result := make(v2.ConfigsPerType)
	response, err := d.client.List(context.TODO())
	if err != nil {
		log.WithFields(field.Type("bucket"), field.Error(err)).Error("Failed to fetch all bucket definitions: %v", err)
		return result, nil
	}

	if !response.IsSuccess() {
		log.WithFields(field.Type("bucket"), field.Error(err)).Error("Failed to fetch all bucket definitions: %s", string(response.Data))
		return result, nil
	}

	configs := d.convertAllObjects(projectName, response.Objects)
	result["bucket"] = configs
	return result, nil
}

func (d *Downloader) convertAllObjects(projectName string, objects [][]byte) []config.Config {
	result := make([]config.Config, 0, len(objects))
	for _, o := range objects {
		var bucketName struct {
			BucketName  string `json:"bucketName"`
			DisplayName string `json:"displayName"`
		}
		if err := json.Unmarshal(o, &bucketName); err != nil {
			return result
		}

		// exclude builtin bucket names
		if strings.HasPrefix(bucketName.BucketName, "dt_") || strings.HasPrefix(bucketName.BucketName, "default_") {
			continue
		}

		// construct config object with generated config ID
		configID := idutils.GenerateUUIDFromString(bucketName.BucketName)
		configCoord := coordinate.Coordinate{
			Project:  projectName,
			Type:     "bucket",
			ConfigId: configID,
		}

		params := map[string]parameter.Parameter{}
		if bucketName.DisplayName != "" {
			params[config.NameParameter] = &value.ValueParameter{Value: bucketName.DisplayName}
		} else {
			params[config.NameParameter] = &value.ValueParameter{Value: bucketName.BucketName} // TODO: clarify!
		}

		originObjectID := fmt.Sprintf("%s_%s", configCoord.Project, configCoord.ConfigId) // TODO: clarify!

		c := config.Config{
			Template:       template.NewDownloadTemplate(configID, configID, string(jsonutils.MarshalIndent(o))),
			Parameters:     params,
			Coordinate:     configCoord,
			Type:           config.BucketType{},
			OriginObjectId: originObjectID,
		}
		result = append(result, c)
	}
	return result
}
