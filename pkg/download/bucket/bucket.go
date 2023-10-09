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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/buckettools"
	jsonutils "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/internal/templatetools"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
)

type skipErr struct {
	msg string
}

func (s skipErr) Error() string {
	return s.msg
}

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

	configs := d.convertAllObjects(projectName, response.All())
	result["bucket"] = configs
	return result, nil
}

func (d *Downloader) convertAllObjects(projectName string, objects [][]byte) []config.Config {
	result := make([]config.Config, 0, len(objects))
	for _, o := range objects {

		c, err := convertObject(o, projectName)
		if err != nil {
			if errors.As(err, &skipErr{}) {
				log.WithFields(field.Type("bucket")).
					Debug("Skipping bucket: %s", err.Error())
			} else {
				log.WithFields(field.Type("bucket"), field.Error(err)).
					Error("Failed to decode API response objects for bucket resource: %v", err)
			}

			continue
		}
		result = append(result, c)
	}

	log.Info("Downloaded %d Grail buckets", len(result))

	return result
}

const (
	bucketName  = "bucketName"
	displayName = "displayName"
	status      = "status"
	version     = "version"
	updatable   = "updatable"
)

func convertObject(o []byte, projectName string) (config.Config, error) {
	r, err := templatetools.NewJSONObject(o)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal bucket: %w", err)
	}

	id, ok := r.Get(bucketName).(string)
	if !ok {
		return config.Config{}, fmt.Errorf("variable %q unreadable", bucketName)
	}

	// skip unmodifiable buckets
	if u, ok := r.Get(updatable).(bool); ok && !u || buckettools.IsDefault(id) {
		return config.Config{}, skipErr{fmt.Sprintf("bucket %q is immutable", id)}
	}

	// buckets that are in the deleting state should not be persisted
	if stat, ok := r.Get(status).(string); ok && stat == "deleting" {
		return config.Config{}, skipErr{fmt.Sprintf("bucket %q is deleting", id)}
	}

	// remove fields that will be set on deployment
	r.Delete(bucketName)
	r.Delete(status)
	r.Delete(version)
	r.Delete(updatable)

	// pull displayName into paramter if one exists
	parameters := map[string]parameter.Parameter{}
	p := r.Parameterize(displayName)
	if p != nil {
		parameters[displayName] = p
	}

	t, err := r.ToJSON()
	if err != nil {
		return config.Config{}, err
	}

	c := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     "bucket",
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.BucketType{},
		Template:       template.NewInMemoryTemplate(id, string(jsonutils.MarshalIndent(t))),
		Parameters:     parameters,
	}

	return c, nil
}
