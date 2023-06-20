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
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client/dtclient"
	config "github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/parameter"
)

func deploySetting(ctx context.Context, settingsClient dtclient.SettingsClient, properties parameter.Properties, renderedConfig string, c *config.Config) (*parameter.ResolvedEntity, error) {
	t, ok := c.Type.(config.SettingsType)
	if !ok {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(c, fmt.Sprintf("config was not of expected type %q, but %q", config.SettingsTypeId, c.Type.ID()))
	}

	scope, err := extractScope(properties)
	if err != nil {
		return &parameter.ResolvedEntity{}, err
	}

	entity, err := settingsClient.UpsertSettings(ctx, dtclient.SettingsObject{
		Coordinate:     c.Coordinate,
		SchemaId:       t.SchemaId,
		SchemaVersion:  t.SchemaVersion,
		Scope:          scope,
		Content:        []byte(renderedConfig),
		OriginObjectId: c.OriginObjectId,
	})
	if err != nil {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(c, err.Error()).withError(err)
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", entity.Id)
	if configName, err := extractConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.Warn("failed to extract name for Settings 2.0 object %q - ID will be used", entity.Id)
	}

	properties[config.IdParameter], err = getEntityID(c, entity)
	if err != nil {
		return &parameter.ResolvedEntity{}, newConfigDeployErr(c, err.Error()).withError(err)
	}

	properties[config.NameParameter] = name

	return &parameter.ResolvedEntity{
		EntityName: name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil

}

func getEntityID(c *config.Config, e dtclient.DynatraceEntity) (string, error) {
	if c.Coordinate.Type == "builtin:management-zones" && featureflags.ManagementZoneSettingsNumericIDs().Enabled() {
		numID, err := idutils.GetNumericIDForObjectID(e.Id)
		if err != nil {
			return "", fmt.Errorf("failed to extract numeric ID for Management Zone Setting with object ID %q: %w", e.Id, err)
		}
		return fmt.Sprintf("%d", numID), nil
	}

	return e.Id, nil
}
