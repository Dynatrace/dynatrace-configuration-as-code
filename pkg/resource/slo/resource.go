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

package slo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/templatetools"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/pointer"
	deployErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project"
)

type DeploySource interface {
	List(ctx context.Context) (api.PagedListResponse, error)
	Update(ctx context.Context, id string, data []byte) (api.Response, error)
	Create(ctx context.Context, data []byte) (api.Response, error)
	Delete(ctx context.Context, id string) (api.Response, error)
}

type Resource struct {
	sloSource DeploySource
	// optional caching
}

func NewDeployAPI(sloSource DeploySource) *Resource {
	return &Resource{sloSource}
}

type sloResponse struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
}

func (d *Resource) Type() string {
	return string(config.ServiceLevelObjectiveID)
}

func (d *Resource) Is(t config.Type) bool {
	_, ok := t.(config.ServiceLevelObjective)
	return ok
}

func (d *Resource) IsDeletePointer(t string) bool {
	return t == string(config.ServiceLevelObjectiveID)
}

func (d *Resource) DeletePriority(_ string) int {
	return 0
}

func (d *Resource) Deploy(ctx context.Context, properties parameter.Properties, renderedConfig string, c *config.Config) (entities.ResolvedEntity, error) {
	ctx = logr.NewContext(ctx, log.WithCtxFields(ctx).GetLogr())
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	externalID := idutils.GenerateExternalID(c.Coordinate)
	requestPayload, err := addExternalIdAndValidate(externalID, renderedConfig)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, "failed to validate slo payload").WithError(err)
	}

	//Strategy 1 when OriginObjectId is set we update the object
	if c.OriginObjectId != "" {
		_, err = d.sloSource.Update(ctx, c.OriginObjectId, requestPayload)
		if err == nil {
			return createResolveEntity(c.OriginObjectId, properties, c), nil
		}

		if !api.IsNotFoundError(err) {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy slo: %s", c.OriginObjectId)).WithError(err)
		}
	}

	//Strategy 2 is to try to find a match with external id and update it
	matchID, match, err := findMatchOnRemote(ctx, d.sloSource, externalID)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("error finding slo with externalID: %s", externalID)).WithError(err)
	}

	if match {
		_, err := d.sloSource.Update(ctx, matchID, requestPayload)
		if err != nil {
			return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to update slo with externalID: %s", externalID)).WithError(err)
		}
		return createResolveEntity(matchID, properties, c), nil
	}

	//Strategy 3 is to create a new slo
	createResponse, err := d.sloSource.Create(ctx, requestPayload)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to deploy slo with externalID: %s", externalID)).WithError(err)
	}

	response, err := responseFromHttpData(createResponse)
	if err != nil {
		return entities.ResolvedEntity{}, deployErrors.NewConfigDeployErr(c, fmt.Sprintf("failed to unmarshal slo with externalID: %s", externalID)).WithError(err)
	}

	return createResolveEntity(response.ID, properties, c), nil
}

func (d *Resource) Preload(_ config.Type) {

}

func (d *Resource) ClearCache() {}

func (d *Resource) Delete(ctx context.Context, entries []pointer.DeletePointer) error {
	errCount := 0
	for _, dp := range entries {
		err := d.deleteSingle(ctx, dp)
		if err != nil {
			log.WithCtxFields(ctx).WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate())).Error("Failed to delete entry: %v", err)
			errCount++
		}
	}
	if errCount > 0 {
		return fmt.Errorf("failed to delete %d %s objects(s)", errCount, config.ServiceLevelObjectiveID)
	}
	return nil
}

func (d *Resource) deleteSingle(ctx context.Context, dp pointer.DeletePointer) error {
	logger := log.WithCtxFields(ctx).WithFields(field.Type(dp.Type), field.Coordinate(dp.AsCoordinate()))

	id := dp.OriginObjectId
	if id == "" {
		var err error
		id, err = d.findEntryWithExternalID(ctx, dp)
		if err != nil {
			return err
		}
	}

	if id == "" {
		logger.Debug("no action needed")
		return nil
	}

	_, err := d.sloSource.Delete(ctx, id)
	if err != nil && !api.IsNotFoundError(err) {
		return fmt.Errorf("failed to delete entry with id '%s': %w", id, err)
	}

	logger.Debug("Config with ID '%s' successfully deleted", id)
	return nil
}

func (d *Resource) findEntryWithExternalID(ctx context.Context, dp pointer.DeletePointer) (string, error) {
	items, err := d.sloSource.List(ctx)
	if err != nil {
		return "", err
	}

	extID := idutils.GenerateExternalID(dp.AsCoordinate())

	var found []entry
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			return "", err
		}
		if e.ExternalID == extID {
			found = append(found, e)
		}
	}

	switch {
	case len(found) == 0:
		return "", nil
	case len(found) > 1:
		var ids []string
		for _, i := range found {
			ids = append(ids, i.ID)
		}
		return "", fmt.Errorf("found more than one %s with same externalId (%s); matching IDs: %s", config.ServiceLevelObjectiveID, extID, ids)
	default:
		return found[0].ID, nil
	}
}

func (d *Resource) DeleteAll(ctx context.Context) error {
	items, err := d.sloSource.List(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			errs = append(errs, err)
			continue
		}
		err := d.deleteSingle(ctx, pointer.DeletePointer{Type: string(config.ServiceLevelObjectiveID), OriginObjectId: e.ID})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (d *Resource) Download(ctx context.Context, projectName string) (project.ConfigsPerType, error) {
	log.Info("Downloading SLO-V2")
	result := project.ConfigsPerType{}
	downloadedConfigs, err := d.sloSource.List(ctx)
	if err != nil {
		log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to fetch the list of existing %s configs: %v", config.ServiceLevelObjectiveID, err)
		// error is ignored
		return nil, nil
	}

	var configs []config.Config
	for _, downloadedConfig := range downloadedConfigs.All() {
		c, err := createConfig(projectName, downloadedConfig)
		if err != nil {
			log.WithFields(field.Type(config.ServiceLevelObjectiveID), field.Error(err)).Error("Failed to convert %s: %v", config.ServiceLevelObjectiveID, err)
			continue
		}
		configs = append(configs, c)
	}
	result[string(config.ServiceLevelObjectiveID)] = configs

	return result, nil
}

func createConfig(projectName string, data []byte) (config.Config, error) {
	jsonObj, err := templatetools.NewJSONObject(data)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	id, ok := jsonObj.Get("id").(string)
	if !ok {
		return config.Config{}, fmt.Errorf("API payload is missing 'id'")
	}

	// delete fields that prevent a re-upload of the configuration
	jsonObj.Delete("id", "version", "externalId")

	jsonRaw, err := jsonObj.ToJSON(true)
	if err != nil {
		return config.Config{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return config.Config{
		Template: template.NewInMemoryTemplate(id, string(jsonRaw)),
		Coordinate: coordinate.Coordinate{
			Project:  projectName,
			Type:     string(config.ServiceLevelObjectiveID),
			ConfigId: id,
		},
		OriginObjectId: id,
		Type:           config.ServiceLevelObjective{},
		Parameters:     make(config.Parameters),
	}, nil
}


type entry struct {
	ID         string `json:"id"`
	ExternalID string `json:"externalId"`
}

func addExternalIdAndValidate(externalId string, renderedConfig string) ([]byte, error) {
	var request map[string]any
	err := json.Unmarshal([]byte(renderedConfig), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to add externalID to slo request payload: %w", err)
	}
	request["externalId"] = externalId
	if _, exists := request["evaluationType"]; exists {
		return nil, errors.New("tried to deploy an slo-v1 configuration to slo-v2")
	}
	return json.Marshal(request)
}

func responseFromHttpData(rawResponse api.Response) (sloResponse, error) {
	var response sloResponse
	err := json.Unmarshal(rawResponse.Data, &response)
	if err != nil {
		return sloResponse{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

func createResolveEntity(id string, properties parameter.Properties, c *config.Config) entities.ResolvedEntity {
	properties[config.IdParameter] = id
	return entities.ResolvedEntity{
		Coordinate: c.Coordinate,
		Properties: properties,
	}
}

func findMatchOnRemote(ctx context.Context, client DeploySource, externalId string) (id string, match bool, err error) {
	apiResponse, err := client.List(ctx)
	if err != nil {
		return "", false, err
	}

	res := sloResponse{}
	for _, raw := range apiResponse.All() {
		if err := json.Unmarshal(raw, &res); err != nil {
			return "", false, err
		}
		if res.ExternalID == externalId {
			return res.ID, true, nil
		}
	}

	return "", false, nil
}
