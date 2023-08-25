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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type Client interface {
	Upsert(ctx context.Context, bucketName string, data []byte) (bucket.Response, error)
}

var _ Client = (*DummyClient)(nil)

type DummyClient struct{}

func (c DummyClient) Upsert(_ context.Context, id string, data []byte) (response bucket.Response, err error) {
	return bucket.Response{
		BucketName: id,
		Data:       data,
	}, nil
}

func Deploy(ctx context.Context, client Client, properties parameter.Properties, renderedConfig string, c *config.Config) (config.ResolvedEntity, error) {
	bucketName := BucketId(c.Coordinate)

	_, err := client.Upsert(ctx, bucketName, []byte(renderedConfig))
	if err != nil {
		return config.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert bucket with bucketName %q", bucketName)).WithError(err)
	}

	properties[config.IdParameter] = bucketName

	return config.ResolvedEntity{
		EntityName: bucketName,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}

// BucketId returns the ID for a bucket based on the coordinate.
// Since the bucket API does not support colons, we concatenate them using underscores.
func BucketId(c coordinate.Coordinate) string {
	return fmt.Sprintf("%s_%s_%s", c.Project, c.Type, c.ConfigId)
}
