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
	"errors"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/buckettools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/attribute"
	escTemplate "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type skipErr struct {
	msg string
}

func (s skipErr) Error() string {
	return s.msg
}

type DownloadSource interface {
	List(ctx context.Context) (buckets.ListResponse, error)
}

type DownloadAPI struct {
	bucketSource DownloadSource
}

func NewDownloadAPI(bucketSource DownloadSource) *DownloadAPI {
	return &DownloadAPI{bucketSource}
}

func (a DownloadAPI) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.InfoContext(ctx, "Downloading Grail buckets")
	result := make(project.ConfigsPerType)
	response, err := a.bucketSource.List(ctx)
	if err != nil {
		log.With(attribute.Type("bucket"), attribute.Error(err)).ErrorContext(ctx, "Failed to fetch all bucket definitions: %v", err)
		return nil, nil
	}

	configs := convertAllObjects(projectName, response.All())
	result["bucket"] = configs
	return result, nil
}

func convertAllObjects(projectName string, objects [][]byte) []config.Config {
	result := make([]config.Config, 0, len(objects))

	lg := log.With(attribute.Type("bucket"))

	for _, o := range objects {

		c, err := convertObject(o, projectName)
		if err != nil {
			if errors.As(err, &skipErr{}) {
				lg.Debug("Skipping bucket: %s", err)
			} else {
				lg.With(attribute.Error(err)).Error("Failed to decode API response objects for bucket resource: %v", err)
			}

			continue
		}
		result = append(result, c)
	}

	lg = lg.With(attribute.Any("configsDownloaded", len(result)))
	switch len(objects) {
	case 0:
		// Info on purpose. Most types have a lot of objects, so skipping printing 'not found' in the default case makes sense. Here it's kept on purpose as bucket is only one type.
		lg.Info("Did not find any buckets to download")
	case len(result):
		lg.Info("Downloaded %d buckets.", len(result))
	default:
		lg.Info("Downloaded %d buckets. Skipped persisting %d unmodifiable bucket(s).", len(result), len(objects)-len(result))
	}

	return result
}

func convertObject(o []byte, projectName string) (config.Config, error) {
	// bucket holds all values we need to check before we persist the object
	type bucket struct {
		Name      string `json:"bucketName"`
		Updatable *bool  `json:"updatable,omitempty"`
		Status    string `json:"status"`
	}
	const (
		bucketName  = "bucketName"
		displayName = "displayName"
		status      = "status"
		version     = "version"
		updatable   = "updatable"
	)

	// escape possible go templates before extracting parameters
	escapedData := escTemplate.UseGoTemplatesForDoubleCurlyBraces(o)
	var b bucket
	if err := json.Unmarshal(escapedData, &b); err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal bucket: %w", err)
	}

	// bucketName acts as the id, thus it must be set
	if b.Name == "" {
		return config.Config{}, fmt.Errorf("bucketName is not set")
	}

	// skip unmodifiable buckets
	if b.Updatable != nil && !*b.Updatable || buckettools.IsDefault(b.Name) {
		return config.Config{}, skipErr{fmt.Sprintf("bucket %q is immutable", b.Name)}
	}

	// buckets that are in the deleting state should not be persisted
	if b.Status == "deleting" {
		return config.Config{}, skipErr{fmt.Sprintf("bucket %q is deleting", b.Name)}
	}

	// remove unnecessary fields
	r, err := templatetools.NewJSONObject(escapedData)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal bucket: %w", err)
	}

	// remove fields that will be set on deployment
	r.Delete(bucketName, status, version, updatable)

	// pull displayName into parameter if one exists
	parameters := map[string]parameter.Parameter{}
	p := r.Parameterize(displayName)
	if p != nil {
		parameters[displayName] = p
	}

	t, err := r.ToJSON(true)
	if err != nil {
		return config.Config{}, err
	}

	c := config.Config{
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.BucketTypeID),
			ConfigId: b.Name,
		},
		OriginObjectId: b.Name,
		Type:           config.BucketType{},
		Template:       template.NewInMemoryTemplate(b.Name, string(t)),
		Parameters:     parameters,
	}

	return c, nil
}
