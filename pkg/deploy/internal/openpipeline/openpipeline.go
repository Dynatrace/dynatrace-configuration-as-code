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

package openpipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/openpipeline"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
)

//go:generate mockgen -source=openpipeline.go -destination=openpipeline_mock.go -package=openpipeline openpipelineClient
type Client interface {
	Update(ctx context.Context, id string, data []byte) (openpipeline.Response, error)
}

func Deploy(ctx context.Context, client Client, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	//create new context to carry logger
	ctx = logr.NewContextWithSlogLogger(ctx, log.WithCtxFields(ctx).SLogger())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	t, ok := c.Type.(config.OpenPipelineType)
	if !ok {
		return entities.ResolvedEntity{}, fmt.Errorf("expected openpipeline config type but found %v", t)
	}

	_, err := client.Update(ctx, t.Kind, []byte(renderedConfig))
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update openpipeline object of kind '%s'", t.Kind)).WithError(err)
	}

	return createResolvedEntity(t.Kind, c.Coordinate, properties), nil
}

func createResolvedEntity(id string, coordinate coordinate.Coordinate, properties parameter.Properties) entities.ResolvedEntity {
	properties[config.IdParameter] = id

	return entities.ResolvedEntity{
		EntityName: id,
		Coordinate: coordinate,
		Properties: properties,
	}
}
