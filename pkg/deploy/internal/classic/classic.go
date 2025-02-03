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
	"strings"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/extract"
)

func Deploy(ctx context.Context, configClient client.ConfigClient, apis api.APIs, properties parameter.Properties, renderedConfig string, conf *config.Config) (entities.ResolvedEntity, error) {
	// create new context to carry logger
	ctx = logr.NewContextWithSlogLogger(ctx, log.WithCtxFields(ctx).SLogger())

	t, ok := conf.Type.(config.ClassicApiType)
	if !ok {
		return entities.ResolvedEntity{}, fmt.Errorf("config was not of expected type %q, but %q", config.ClassicApiTypeID, conf.Type.ID())
	}

	apiToDeploy, found := apis[t.Api]
	if !found {
		return entities.ResolvedEntity{}, fmt.Errorf("unknown api `%s`. this is most likely a bug", t.Api)
	}

	if apiToDeploy.HasParent() {
		scope, err := extract.Scope(properties)
		if err != nil {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to extract scope for config %q", conf.Type.ID())
		}
		apiToDeploy = apiToDeploy.ApplyParentObjectID(scope)
	}

	configName := ""
	var err error
	if t.Api != api.DashboardShareSettings {
		configName, err = extract.ConfigName(conf, properties)
		if err != nil {
			return entities.ResolvedEntity{}, err
		}
	}

	if apiToDeploy.DeprecatedBy != "" {
		log.WithCtxFields(ctx).Warn("API for \"%s\" is deprecated! Please consider migrating to \"%s\"!", apiToDeploy.ID, apiToDeploy.DeprecatedBy)
	}

	var dtEntity dtclient.DynatraceEntity
	if apiToDeploy.NonUniqueName {
		dtEntity, err = upsertNonUniqueNameConfig(ctx, configClient, apiToDeploy, conf, configName, renderedConfig)
	} else {
		dtEntity, err = configClient.UpsertByName(ctx, apiToDeploy, configName, []byte(renderedConfig))
	}

	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(conf, err.Error()).WithError(err)
	}

	properties[config.IdParameter] = dtEntity.Id
	properties[config.NameParameter] = dtEntity.Name

	return entities.ResolvedEntity{
		EntityName: dtEntity.Name,
		Coordinate: conf.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil
}

func upsertNonUniqueNameConfig(ctx context.Context, client client.ConfigClient, apiToDeploy api.API, conf *config.Config, configName string, renderedConfig string) (dtclient.DynatraceEntity, error) {
	configID := conf.Coordinate.ConfigId
	projectId := conf.Coordinate.Project

	entityUuid := configID

	isUUIDOrMeID := idutils.IsUUID(entityUuid) || idutils.IsMeId(entityUuid)
	if !isUUIDOrMeID {
		entityUuid = idutils.GenerateUUIDFromConfigId(projectId, configID)
	}

	// for now I only use the origin object id (if set) as entityUuid for "user-action-and-session-properties-mobile",
	// as i am not sure what side effects it will have it is occasionally set for others as well.
	if apiToDeploy.ID == api.UserActionAndSessionPropertiesMobile {
		if conf.OriginObjectId != "" {
			entityUuid = conf.OriginObjectId
		} else {
			// if we didn't got an origin object id from a download, lets use the generated entity id,
			// however "user-action-and-session-properties-mobile" ids (keys) don't allow "-" and must be lowercase
			entityUuid = strings.ReplaceAll(entityUuid, "-", "")
			entityUuid = strings.ToLower(entityUuid)
		}
	}

	// check if we are dealing with a non-unique name configuration that appears multiple times
	// in a monaco project. if that's the case, we need to handle it differently, by setting the
	// duplicate parameter accordingly
	var duplicate bool
	if val, exists := conf.Parameters[config.NonUniqueNameConfigDuplicationParameter]; exists {
		resolvedVal, err := val.ResolveValue(parameter.ResolveContext{})
		if err != nil {
			return dtclient.DynatraceEntity{}, err
		}
		resolvedValBool, ok := resolvedVal.(bool)
		if !ok {
			return dtclient.DynatraceEntity{}, err
		}
		duplicate = resolvedValBool
	}
	return client.UpsertByNonUniqueNameAndId(ctx, apiToDeploy, entityUuid, configName, []byte(renderedConfig), duplicate)
}
