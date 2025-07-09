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

package bucket

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

type DeploySource interface {
	Get(ctx context.Context, bucketName string) (api.Response, error)
	Create(ctx context.Context, bucketName string, data []byte) (api.Response, error)
	Update(ctx context.Context, bucketName string, data []byte) (api.Response, error)
}

type DeployAPI struct {
	source DeploySource
}

func NewDeployAPI(source DeploySource) *DeployAPI {
	return &DeployAPI{source}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	var bucketName string

	if c.OriginObjectId != "" {
		bucketName = c.OriginObjectId
	} else {
		bucketName = idutils.GenerateBucketName(c.Coordinate)
	}

	// create new context to carry logger
	ctx = logr.NewContextWithSlogLogger(ctx, slog.Default())
	err := d.upsert(ctx, bucketName, []byte(renderedConfig))
	if err != nil {
		var apiErr api.APIError
		if errors.As(err, &apiErr) {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to upsert bucket '%s': %w", bucketName, err)
		}

		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert bucket '%s'", bucketName)).WithError(err)
	}

	properties[config.IdParameter] = bucketName

	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}

func (d DeployAPI) upsert(ctx context.Context, bucketName string, data []byte) error {
	// Check the status of a bucket (updating/creating/deleting) and if it even exists
	if bucketExists, err := buckets.AwaitActiveOrNotFound(ctx, d.source, bucketName, maxRetryDuration, durationBetweenRetries); err != nil {
		return err
	} else if bucketExists {
		_, err := d.source.Update(ctx, bucketName, data)
		return err
	}

	if _, err := d.source.Create(ctx, bucketName, data); err != nil {
		return err
	}
	// after create wait for bucket being active/deleted
	start := time.Now()
	if bucketExists, err := buckets.AwaitActiveOrNotFound(ctx, d.source, bucketName, maxRetryDuration, durationBetweenRetries); err != nil {
		return err
	} else if bucketExists {
		log.DebugContext(ctx, "Bucket '%s' became active and is ready to use", bucketName)
	}
	// wait until bucket cache refreshes, so that other calls don't have any problems
	elapsed := time.Since(start)
	if elapsed >= time.Minute {
		return nil
	}

	time.Sleep(time.Minute - elapsed)
	return nil
}
