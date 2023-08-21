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

package deploy

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

type bucketClient interface {
	Upsert(ctx context.Context, id string, data []byte) (bucket.Response, error)
}

var _ bucketClient = (*bucket.Client)(nil)

func deployBucket(ctx context.Context, client bucketClient, properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, error) {
	id := BucketId(c.Coordinate)

	_, err := client.Upsert(ctx, id, []byte(renderedConfig))
	if err != nil {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(c, fmt.Sprintf("failed to upsert bucket with id %q", id)).withError(err)
	}

	properties[config.IdParameter] = id

	return &parameter.ResolvedEntity{
		EntityName: id,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}

// BucketId returns the ID for a bucket based on the coordinate.
// Since the bucket API does not support colons, we concatenate them using underscores.
func BucketId(c coordinate.Coordinate) string {
	return fmt.Sprintf("%s_%s_%s", c.Project, c.Type, c.ConfigId)
}
