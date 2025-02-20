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
	"strings"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
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

func Deploy(ctx context.Context, settingsClient client.SettingsClient, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	t, ok := c.Type.(config.SettingsType)
	if !ok {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, fmt.Sprintf("config was not of expected type %q, but %q", config.SettingsTypeID, c.Type.ID()))
	}

	insertAfter, err := getAndParseInsertAfterParameter(properties)
	if err != nil {
		return entities.ResolvedEntity{}, err
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

	insertOptions := dtclient.UpsertSettingsOptions{
		OverrideRetry: nil,
		InsertAfter:   insertAfter,
	}

	if c.HasRefTo(string(config.BucketTypeID)) {
		insertOptions.OverrideRetry = &dtclient.RetrySetting{WaitTime: 10 * time.Second, MaxRetries: 6}
	}

	if c.HasRefTo(api.ApplicationWeb) {
		insertOptions.OverrideRetry = &dtclient.DefaultRetrySettings.VeryLong
	}

	dtEntity, err := settingsClient.Upsert(ctx, settingsObj, insertOptions)
	if err != nil {
		return entities.ResolvedEntity{}, errors.NewConfigDeployErr(c, err.Error()).WithError(err)
	}

	name := fmt.Sprintf("[UNKNOWN NAME]%s", dtEntity.Id)
	if configName, err := extract.ConfigName(c, properties); err == nil {
		name = configName
	} else {
		log.WithCtxFields(ctx).Debug("failed to extract name for Settings 2.0 object %q - ID will be used", dtEntity.Id)
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

// getAndParseInsertAfterParameter finds and parses the `insertAfter parameter.
//
//   - null is returned if the position is not set
//   - dtclient.InsertPositionFront is returned iff the position is set to `front` (case-insensitive)
//   - dtclient.InsertPositionBack is returned iff the position is set to `back` (case-insensitive)
//   - otherwise, the value itself is returned (id of another config)
//
// It returns an error if the insertAfter parameter is something other than an error (e.g. compound parameter).
func getAndParseInsertAfterParameter(properties parameter.Properties) (*string, error) {
	param, found := properties[config.InsertAfterParameter]
	if !found {
		return nil, nil
	}

	insertAfter, ok := param.(string)
	if !ok {
		return nil, fmt.Errorf("'insertAfter' parameter must be a string of either an ID, '%s', or '%s', got '%v'", dtclient.InsertPositionFront, dtclient.InsertPositionBack, param)
	}

	// Test if insertAfter are magic values (case-insensitive)
	// We can't modify the original insertAfter value, as IDs must not be upper-cased.
	switch strings.ToUpper(insertAfter) {
	case dtclient.InsertPositionFront:
		insertAfter = dtclient.InsertPositionFront
	case dtclient.InsertPositionBack:
		insertAfter = dtclient.InsertPositionBack
	}

	return &insertAfter, nil
}

func getEntityID(c *config.Config, e dtclient.DynatraceEntity) (string, error) {
	if c.Coordinate.Type == "builtin:management-zones" && featureflags.ManagementZoneSettingsNumericIDs.Enabled() {
		numID, err := idutils.GetNumericIDForObjectID(e.Id)
		if err != nil {
			return "", fmt.Errorf("failed to extract numeric ID for Management Zone Setting with object ID %q: %w", e.Id, err)
		}
		return fmt.Sprintf("%d", numID), nil
	}

	return e.Id, nil
}
