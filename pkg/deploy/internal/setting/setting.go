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

package setting

import (
	"context"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/internal/extract"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/events"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"time"
)

func Deploy(ctx context.Context, settingsClient client.SettingsClient, properties parameter.Properties, renderedConfig string, c *config.Config, insertAfter string) (entities.ResolvedEntity, error) {
	t, ok := c.Type.(config.SettingsType)
	if !ok {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("config was not of expected type %q, but %q", config.SettingsTypeId, c.Type.ID()))
	}

	scope, err := extract.Scope(properties)
	if err != nil {
		return entities.ResolvedEntity{}, err
	}

	settingsObj := dtclient.SettingsObject{
		Coordinate:     c.Coordinate,
		SchemaId:       t.SchemaId,
		SchemaVersion:  t.SchemaVersion,
		Scope:          scope,
		Content:        []byte(renderedConfig),
		OriginObjectId: c.OriginObjectId,
	}
	upsertOptions := makeUpsertOptions(c, insertAfter)

	dtEntity, err := settingsClient.UpsertSettings(ctx, settingsObj, upsertOptions)
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, err.Error()).WithError(err)
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", dtEntity.Id)
	if configName, err := extract.ConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.WithCtxFields(ctx).Debug("failed to extract name for Settings 2.0 object %q - ID will be used", dtEntity.Id)
		events.NewFromContextOrDiscard(ctx).Send(events.ConfigDeploymentLogEvent{
			InternalEvent: events.NewInternalEventNow(c.Coordinate.String()),
			Type:          "WARN",
			Message:       fmt.Sprintf("failed to extract name for Settings 2.0 object %q - ID will be used", dtEntity.Id),
		})
	}

	properties[config.IdParameter], err = getEntityID(c, dtEntity)
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, err.Error()).WithError(err)
	}

	properties[config.NameParameter] = name

	return entities.ResolvedEntity{
		EntityName: name,
		Coordinate: c.Coordinate,
		Properties: properties,
		Skip:       false,
	}, nil

}

func makeUpsertOptions(c *config.Config, insertAfter string) dtclient.UpsertSettingsOptions {
	// SPECIAL HANDLING: if settings config to be deployed has a reference to a "bucket" definition
	// we need to drastically increase the retry settings for the upsert operation, as it could take
	// up to 1 minute until the operation succeeds in case a bucket was just created before
	var hasRefToBucket bool
	refs := c.References()
	for _, r := range refs {
		if r.Type == "bucket" {
			hasRefToBucket = true
		}
	}
	upsertOpts := dtclient.UpsertSettingsOptions{
		InsertAfter: insertAfter,
	}
	if hasRefToBucket {
		upsertOpts.OverrideRetry = &rest.RetrySetting{
			WaitTime:   10 * time.Second,
			MaxRetries: 6,
		}
	}
	return upsertOpts
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
