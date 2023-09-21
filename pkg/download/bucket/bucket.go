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
	tools "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/buckettools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/raw"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
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
func (d *Downloader) Download(projectName string, _ ...config.BucketType) (v2.ConfigsPerType, error) { // error in return is just to complain to interface
	result := make(v2.ConfigsPerType)
	response, err := d.client.List(context.TODO())
	if err != nil {
		log.WithFields(field.Type("bucket"), field.Error(err)).Error("Failed to fetch all bucket definitions: %v", err)
		return nil, nil
	}

	if apiErr, isErr := response.AsAPIError(); isErr {
		log.WithFields(field.Type("bucket"), field.Error(apiErr)).Error("Failed to fetch all bucket definitions: %v", apiErr)
		return nil, nil
	}

	configs := d.convertAllObjects(projectName, response.Objects)
	result["bucket"] = configs
	return result, nil
}

func (d *Downloader) convertAllObjects(projectName string, objects [][]byte) []config.Config {
	result := make([]config.Config, 0, len(objects))
	for _, o := range objects {

		c, err := convertObject(o, projectName)
		if err != nil {
			log.WithFields(field.Coordinate(coordinate.Coordinate{Project: c.Coordinate.Project, Type: "bucket", ConfigId: c.Coordinate.ConfigId}), field.Error(err)).
				Warn("Failed to get configuration for %v (%s): %v", c.Coordinate.ConfigId, "bucket", err)
			continue
		}

		if c == nil {
			continue
		}
		result = append(result, *c)
	}

	log.Info("Downloaded %d Grail buckets", len(result))

	return result
}

const (
	bucketName  = "bucketName"
	displayName = "displayName"
)

func convertObject(o []byte, projectName string) (*config.Config, error) {
	c := config.Config{
		Coordinate: coordinate.Coordinate{
			Project: projectName,
			Type:    "bucket",
		},
		Type: config.BucketType{},
	}

	r, err := raw.New(o)
	if err != nil {
		return &c, err
	}

	id, ok := r.Get(bucketName).(string)
	if !ok {
		return &c, fmt.Errorf("variable %q unreadable", bucketName)
	}

	// exclude builtin bucket names
	if tools.IsDefault(id) {
		return nil, nil
	}

	// construct config object with generated config ID
	configID := idutils.GenerateUUIDFromString(id)
	c.Coordinate = coordinate.Coordinate{
		Project:  projectName,
		Type:     "bucket",
		ConfigId: configID,
	}

	c.OriginObjectId = r.Get(bucketName).(string)

	r.Delete(bucketName)

	c.Parameters = map[string]parameter.Parameter{}
	p := r.Parameterize(displayName)
	if p != nil {
		c.Parameters[displayName] = p
	}

	t, err := r.ToJSON()
	if err != nil {
		return &c, err
	}
	c.Template = template.NewDownloadTemplate(configID, configID, string(jsonutils.MarshalIndent(t)))

	return &c, nil
}

func getValueForAttribute(raw []byte, name string) (string, error) {
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return "", err
	}
	if m[name] != nil {
		return fmt.Sprintf("%v", m[name]), nil

	}
	return "", nil
}

func replaceAttributeWith(raw []byte, attributeName, value string) ([]byte, error) {
	var m map[string]any
	err := json.Unmarshal(raw, &m)
	if err != nil {
		return raw, err
	}
	if _, exits := m[attributeName]; exits {
		m[attributeName] = "{{." + value + "}}"
	}

	modified, err := json.Marshal(m)
	if err != nil {
		return raw, err
	}

	return modified, nil
}
