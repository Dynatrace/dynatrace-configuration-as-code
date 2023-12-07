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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	clientErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/go-logr/logr"
	"net/http"
)

type Client interface {
	Upsert(ctx context.Context, bucketName string, data []byte) (buckets.Response, error)
}

var _ Client = (*DummyClient)(nil)

type DummyClient struct{}

func (c DummyClient) Upsert(_ context.Context, id string, data []byte) (response buckets.Response, err error) {
	return buckets.Response{
		StatusCode: http.StatusOK,
		Data:       data,
	}, nil
}

func Deploy(ctx context.Context, client Client, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	var bucketName string

	if c.OriginObjectId != "" {
		bucketName = c.OriginObjectId
	} else {
		bucketName = idutils.GenerateBucketName(c.Coordinate)
	}

	// create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	resp, err := client.Upsert(ctx, bucketName, []byte(renderedConfig))
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert bucket with bucketName %q", bucketName)).WithError(err)
	}
	if !resp.IsSuccess() {
		return entities.ResolvedEntity{}, clientErrors.NewRespErr(fmt.Sprintf("failed to upsert bucket with bucketName %q", bucketName), clientErrors.Response{Body: resp.Data, StatusCode: resp.StatusCode})
	}

	properties[config.IdParameter] = bucketName

	return entities.ResolvedEntity{
		EntityName: bucketName,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}
