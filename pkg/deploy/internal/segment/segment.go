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
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	segment "github.com/dynatrace/dynatrace-configuration-as-code-core/clients/grailfiltersegments"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/go-logr/logr"
	"net/http"
	"time"
)

type DeploySegmentClient interface {
	Upsert(ctx context.Context, id string, data []byte) (segment.Response, error)
	Get(ctx context.Context, id string) (segment.Response, error)
}

func Deploy(ctx context.Context, client DeploySegmentClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	externalId := c.Coordinate.String()
	println(externalId) //@TODO remove this, as its only here for debug

	//create new context to carry logger
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	//Strategy 1 if OriginObjectId is set check on remote if an object can be found and update it,
	//also update the externalId to the computed one we generate
	if c.OriginObjectId != "" {
		getResponse, err := client.Get(ctx, c.OriginObjectId)
		if err != nil {
			return entities.ResolvedEntity{}, err
		}
		if getResponse.StatusCode != http.StatusNotFound {
			//@TODO how to set externalId here? unmarshall and set then marshall
			_, err := client.Upsert(ctx, c.OriginObjectId, []byte(renderedConfig))
			if err != nil {
				//println(updateResponse)
			}
		}
	}

	//Strategy 2 is to generate external id and either update or create object
	_, err := client.Upsert(ctx, externalId, []byte(renderedConfig))
	if err != nil {
		var apiErr api.APIError
		if errors.As(err, &apiErr) {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to upsert segment with segmentName %q: %w", externalId, err)
		}

		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to upsert segment with segmentName %q", externalId)).WithError(err)
	}
	// Set this to the upserte id returned by api
	properties[config.IdParameter] = externalId

	return entities.ResolvedEntity{
		EntityName: externalId,
		Coordinate: c.Coordinate,
		Properties: properties,
	}, nil
}
