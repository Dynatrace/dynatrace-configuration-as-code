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

package classic

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/resource/extract"
)

type DeploySource interface {
	UpsertByName(ctx context.Context, a api.API, name string, payload []byte) (dtclient.DynatraceEntity, error)
	UpsertByNonUniqueNameAndId(ctx context.Context, a api.API, entityID string, name string, payload []byte, duplicate bool) (dtclient.DynatraceEntity, error)
}

type DeployAPI struct {
	source DeploySource
	apis   api.APIs
}

func NewDeployAPI(source DeploySource, apis api.APIs) *DeployAPI {
	return &DeployAPI{source, apis}
}

func (d DeployAPI) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, conf *config.Config) (entities.ResolvedEntity, error) {
	// create new context to carry logger
	ctx = logr.NewContextWithSlogLogger(ctx, slog.Default())

	t, ok := conf.Type.(config.ClassicApiType)
	if !ok {
		return entities.ResolvedEntity{}, fmt.Errorf("config was not of expected type '%s', but '%s'", config.ClassicApiTypeID, conf.Type.ID())
	}

	apiToDeploy, found := d.apis[t.Api]
	if !found {
		return entities.ResolvedEntity{}, fmt.Errorf("unknown API '%s'. this is most likely a bug", t.Api)
	}

	if apiToDeploy.HasParent() {
		scope, err := extract.Scope(properties)
		if err != nil {
			return entities.ResolvedEntity{}, fmt.Errorf("failed to extract scope for config '%s': %w", conf.Type.ID(), err)
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

	var dtEntity dtclient.DynatraceEntity
	if apiToDeploy.NonUniqueName {
		dtEntity, err = d.upsertNonUniqueNameConfig(ctx, apiToDeploy, conf, configName, renderedConfig)
	} else {
		dtEntity, err = d.source.UpsertByName(ctx, apiToDeploy, configName, []byte(renderedConfig))
	}

	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(conf, err.Error()).WithError(err)
	}

	properties[config.IdParameter] = dtEntity.Id
	properties[config.NameParameter] = dtEntity.Name

	return entities.ResolvedEntity{
		Coordinate: conf.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil
}

func (d DeployAPI) upsertNonUniqueNameConfig(ctx context.Context, apiToDeploy api.API, conf *config.Config, configName string, renderedConfig string) (dtclient.DynatraceEntity, error) {
	duplicate, err := checkIsDuplicate(conf.Parameters)
	if err != nil {
		return dtclient.DynatraceEntity{}, err
	}

	entityUUID := conf.Coordinate.ConfigId
	isUUIDOrMeID := idutils.IsUUID(entityUUID) || idutils.IsMeId(entityUUID)

	if !isUUIDOrMeID {
		entityUUID = idutils.GenerateUUIDFromConfigId(conf.Coordinate.Project, entityUUID)
	}

	return d.source.UpsertByNonUniqueNameAndId(ctx, apiToDeploy, entityUUID, configName, []byte(renderedConfig), duplicate)
}

// checkIsDuplicate checks if we are dealing with a non-unique name configuration that appears multiple times
// in a monaco project. if that's the case, we need to handle it differently, by setting the duplicate parameter accordingly
func checkIsDuplicate(parameters config.Parameters) (bool, error) {
	if val, exists := parameters[config.NonUniqueNameConfigDuplicationParameter]; exists {
		resolvedVal, err := val.ResolveValue(parameter.ResolveContext{})
		if err != nil {
			return false, err
		}
		resolvedValBool, ok := resolvedVal.(bool)
		if !ok {
			return false, fmt.Errorf("invalid boolean value for '%s', got '%T'", config.NonUniqueNameConfigDuplicationParameter, resolvedVal)
		}
		return resolvedValBool, nil
	}
	return false, nil
}
