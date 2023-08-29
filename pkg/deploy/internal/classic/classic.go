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

package classic

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/extract"
)

func Deploy(ctx context.Context, configClient dtclient.ConfigClient, apis api.APIs, properties parameter.Properties, renderedConfig string, conf *config.Config) (config.ResolvedEntity, error) {
	t, ok := conf.Type.(config.ClassicApiType)
	if !ok {
		return config.ResolvedEntity{}, fmt.Errorf("config was not of expected type %q, but %q", config.ClassicApiTypeId, conf.Type.ID())
	}

	apiToDeploy, found := apis[t.Api]
	if !found {
		return config.ResolvedEntity{}, fmt.Errorf("unknown api `%s`. this is most likely a bug", t.Api)
	}

	configName, err := extract.ConfigName(conf, properties)
	if err != nil {
		return config.ResolvedEntity{}, err
	}

	if apiToDeploy.DeprecatedBy != "" {
		log.WithCtxFields(ctx).Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", apiToDeploy.ID, apiToDeploy.DeprecatedBy)
	}

	var entity dtclient.DynatraceEntity
	if apiToDeploy.NonUniqueName {
		entity, err = upsertNonUniqueNameConfig(ctx, configClient, apiToDeploy, conf, configName, renderedConfig)
	} else {
		entity, err = configClient.UpsertConfigByName(ctx, apiToDeploy, configName, []byte(renderedConfig))
	}

	if err != nil {
		return config.ResolvedEntity{}, errors.NewConfigDeployErr(conf, err.Error()).WithError(err)
	}

	properties[config.IdParameter] = entity.Id
	properties[config.NameParameter] = entity.Name

	return config.ResolvedEntity{
		EntityName: entity.Name,
		Coordinate: conf.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil
}

func upsertNonUniqueNameConfig(ctx context.Context, client dtclient.ConfigClient, apiToDeploy api.API, conf *config.Config, configName string, renderedConfig string) (dtclient.DynatraceEntity, error) {
	configID := conf.Coordinate.ConfigId
	projectId := conf.Coordinate.Project

	entityUuid := configID

	isUUIDOrMeID := idutils.IsUUID(entityUuid) || idutils.IsMeId(entityUuid)
	if !isUUIDOrMeID {
		entityUuid = idutils.GenerateUUIDFromConfigId(projectId, configID)
	}

	return client.UpsertConfigByNonUniqueNameAndId(ctx, apiToDeploy, entityUuid, configName, []byte(renderedConfig))
}
