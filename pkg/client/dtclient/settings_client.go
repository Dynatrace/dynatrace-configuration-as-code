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

package dtclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/filter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

type (
	// SettingsObject contains all the information necessary to create/update a settings object
	SettingsObject struct {
		// Coordinate holds all the information for Monaco to identify a settings object
		Coordinate coordinate.Coordinate
		// SchemaId is the Dynatrace settings schema ID
		SchemaId,
		// SchemaVersion is the version of the schema
		SchemaVersion,
		// Scope is the scope of the schema
		Scope string
		// Content is the rendered config for the given settings object
		Content []byte
		// OriginObjectId is the object id of the Settings object when it was downloaded from an environment
		OriginObjectId string
	}

	Schema struct {
		SchemaId         string
		Ordered          bool
		UniqueProperties [][]string
	}

	SchemaList []struct {
		SchemaId string `json:"schemaId"`
		Ordered  bool   `json:"ordered"`
	}

	// SchemaListResponse is the response type returned by the ListSchemas operation
	SchemaListResponse struct {
		Items      SchemaList `json:"items"`
		TotalCount int        `json:"totalCount"`
	}

	postResponse struct {
		ObjectId string `json:"objectId"`
	}

	settingsRequest struct {
		SchemaId      string `json:"schemaId"`
		ExternalId    string `json:"externalId,omitempty"`
		Scope         string `json:"scope"`
		Value         any    `json:"value"`
		SchemaVersion string `json:"schemaVersion,omitempty"`
		ObjectId      string `json:"objectId,omitempty"`
		InsertAfter   string `json:"insertAfter,omitempty"`
	}

	schemaConstraint struct {
		Type             string   `json:"type"`
		UniqueProperties []string `json:"uniqueProperties"`
	}

	// schemaDetailsResponse is the response type returned by the getSchema operation
	schemaDetailsResponse struct {
		SchemaId          string             `json:"schemaId"`
		Ordered           bool               `json:"ordered"`
		SchemaConstraints []schemaConstraint `json:"schemaConstraints"`
	}
)

func (d *SettingsClient) CacheSettings(ctx context.Context, schemaID string) error {
	_, err := d.ListSettings(ctx, schemaID, ListSettingsOptions{})
	return err
}

func (d *SettingsClient) ListSchemas(ctx context.Context) (schemas SchemaList, err error) {
	queryParams := url.Values{}
	queryParams.Add("fields", "ordered,schemaId")

	// getting all schemas does not have pagination
	resp, err := coreapi.AsResponseOrError(d.client.GET(ctx, d.settingsSchemaAPIPath, corerest.RequestOptions{QueryParams: queryParams, CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		return nil, fmt.Errorf("failed to GET schemas: %w", err)
	}

	var result SchemaListResponse
	err = json.Unmarshal(resp.Data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema list: %w", err)
	}

	if result.TotalCount != len(result.Items) {
		log.Warn("Total count of settings 2.0 schemas (=%d) does not match with count of actually downloaded settings 2.0 schemas (=%d)", result.TotalCount, len(result.Items))
	}

	return result.Items, nil
}

func (d *SettingsClient) GetSchemaById(ctx context.Context, schemaID string) (constraints Schema, err error) {
	if ret, cached := d.schemaCache.Get(schemaID); cached {
		return ret, nil
	}

	ret := Schema{SchemaId: schemaID}
	endpoint := d.settingsSchemaAPIPath + "/" + schemaID
	r, err := coreapi.AsResponseOrError(d.client.GET(ctx, endpoint, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		return Schema{}, err
	}

	var sd schemaDetailsResponse
	err = json.Unmarshal(r.Data, &sd)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to unmarshal schema %q: %w", schemaID, err)
	}

	for _, sc := range sd.SchemaConstraints {
		if sc.Type == "UNIQUE" {
			ret.UniqueProperties = append(ret.UniqueProperties, sc.UniqueProperties)
		}
	}
	ret.Ordered = sd.Ordered

	d.schemaCache.Set(schemaID, ret)
	return ret, nil
}

// handleUpsertUnsupportedVersion implements special handling for updating settings 2.0 objects on tenants with version pre 1.262.0
// Tenants with versions < 1.262 are not able to handle updates of existing
// settings 2.0 objects that are non-deletable.
// So we check if the object with originObjectID already exists, if yes and the tenant is older than 1.262
// then we cannot perform the upsert operation
func (d *SettingsClient) handleUpsertUnsupportedVersion(ctx context.Context, obj SettingsObject) (DynatraceEntity, error) {

	fetchedSettingObj, err := d.GetSettingById(ctx, obj.OriginObjectId)
	if err != nil {
		apiErr := coreapi.APIError{}
		// Settings API returns 400 StatusBadRequest for 404 StatusNotFound
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest || apiErr.StatusCode == http.StatusNotFound {
			return DynatraceEntity{}, fmt.Errorf("unable to fetch settings object with object id %q: %w", obj.OriginObjectId, err)
		}
	}

	log.WithCtxFields(ctx).Warn("Unable to update Settings 2.0 object of schema %q and object id %q on Dynatrace environment with a version < 1.262.0", obj.SchemaId, obj.OriginObjectId)
	return DynatraceEntity{
		Id:   fetchedSettingObj.ObjectId,
		Name: fetchedSettingObj.ObjectId,
	}, nil

}

func (d *SettingsClient) UpsertSettings(ctx context.Context, obj SettingsObject, upsertOptions UpsertSettingsOptions) (result DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		result, err = d.upsertSettings(ctx, obj, upsertOptions)
	})
	return
}

func (d *SettingsClient) upsertSettings(ctx context.Context, obj SettingsObject, upsertOptions UpsertSettingsOptions) (result DynatraceEntity, err error) {
	if !d.serverVersion.Invalid() && d.serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262, Patch: 0}) {
		return d.handleUpsertUnsupportedVersion(ctx, obj)
	}

	if matchingObject, found, err := d.findObjectWithMatchingConstraints(ctx, obj); err != nil {
		return DynatraceEntity{}, err
	} else if found {

		var props []string
		for k, v := range matchingObject.matches {
			props = append(props, fmt.Sprintf("(%v = %v)", k, v))
		}

		log.WithCtxFields(ctx).Debug("Updating existing object %q with matching unique properties: %v", matchingObject.object.ObjectId, strings.Join(props, ", "))
		obj.OriginObjectId = matchingObject.object.ObjectId
	}

	// generate legacy external ID without project name.
	// and check if settings object with that external ID exists
	// This exists for avoiding breaking changes when we enhanced external id generation with full coordinates (incl. project name)
	// This can be removed in a later release of monaco
	legacyExternalID, err := d.generateExternalID(coordinate.Coordinate{Type: obj.Coordinate.Type, ConfigId: obj.Coordinate.ConfigId})
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("unable to generate external id: %w", err)
	}

	settingsWithExternalID, err := d.ListSettings(ctx, obj.SchemaId, ListSettingsOptions{
		Filter: func(object DownloadSettingsObject) bool { return object.ExternalId == legacyExternalID },
	})
	if err != nil {
		return DynatraceEntity{}, err
	}

	if len(settingsWithExternalID) > 0 {
		obj.OriginObjectId = settingsWithExternalID[0].ObjectId
	}

	externalID, err := d.generateExternalID(obj.Coordinate)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("unable to generate external id: %w", err)
	}

	// If the server contains two configs, one with the origin-object-id and a second config with the externalID,
	// it is not possible to update the setting using the externalId and origin-object-id on the same POST request,
	// as two settings objects can be the target of the change. In this case, we remove the origin-object-id
	// and only update the object using the externalId.
	settings, err := d.ListSettings(ctx, obj.SchemaId, ListSettingsOptions{
		Filter: func(object DownloadSettingsObject) bool {
			return object.ObjectId == obj.OriginObjectId || object.ExternalId == externalID
		},
	})
	if err != nil {
		return DynatraceEntity{}, err
	}
	if len(settings) == 2 {
		var exIdSetting, ooIdSetting string
		if settings[0].ExternalId == externalID {
			exIdSetting = settings[0].ObjectId
			ooIdSetting = settings[1].ObjectId
		} else {
			exIdSetting = settings[1].ObjectId
			ooIdSetting = settings[0].ObjectId
		}

		log.WithCtxFields(ctx).Warn("Found two configs, one with the defined originObjectId (%q), and one with the expected monaco externalId (%q). Updating the one with the expected externalId.", ooIdSetting, exIdSetting)
		obj.OriginObjectId = ""
	}

	if schema, ok := d.schemaCache.Get(obj.SchemaId); ok {
		if upsertOptions.InsertAfter != "" && !schema.Ordered {
			return DynatraceEntity{}, fmt.Errorf("'%s' is not an ordered setting, hence 'insertAfter' is not supported for this type of setting object", obj.SchemaId)
		}
	}

	payload, err := buildPostRequestPayload(ctx, obj, externalID, upsertOptions.InsertAfter)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to build settings object: %w", err)
	}

	retrySetting := d.retrySettings.Normal
	if upsertOptions.OverrideRetry != nil {
		retrySetting = *upsertOptions.OverrideRetry
	}

	resp, err := SendWithRetryWithInitialTry(ctx, d.client.POST, d.settingsObjectAPIPath, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}, payload, retrySetting)
	if err != nil {
		d.settingsCache.Delete(obj.SchemaId)
		return DynatraceEntity{}, fmt.Errorf("failed to create or update Settings object with externalId %s: %w", externalID, err)
	}

	entity, err := parsePostResponse(resp.Data)
	if err != nil {
		return DynatraceEntity{}, err
	}

	log.WithCtxFields(ctx).Debug("Created/Updated object %s (%s) with externalId %s", obj.Coordinate.ConfigId, obj.SchemaId, externalID)
	return entity, nil
}

type match struct {
	object  DownloadSettingsObject
	matches constraintMatch
}

func (d *SettingsClient) findObjectWithMatchingConstraints(ctx context.Context, source SettingsObject) (match, bool, error) {
	constraints, err := d.GetSchemaById(ctx, source.SchemaId)
	if err != nil {
		return match{}, false, fmt.Errorf("unable to get details for schema %q: %w", source.SchemaId, err)
	}

	if len(constraints.UniqueProperties) == 0 {
		return match{}, false, nil
	}

	objects, err := d.ListSettings(ctx, source.SchemaId, ListSettingsOptions{})
	if err != nil {
		return match{}, false, fmt.Errorf("unable to get existing settings objects for %q schema: %w", source.SchemaId, err)
	}

	target, found, err := findObjectWithSameConstraints(constraints, source, objects)
	if err != nil {
		return match{}, false, err
	}
	return target, found, nil
}

func findObjectWithSameConstraints(schema Schema, source SettingsObject, objects []DownloadSettingsObject) (match, bool, error) {
	candidates := make(map[int]constraintMatch)

	for _, uniqueKeys := range schema.UniqueProperties {
		for j, o := range objects {
			if o.Scope != source.Scope {
				continue // settings in different Scopes can't be the same
			}

			matchFound, constraintMatches, err := doObjectsMatchBasedOnUniqueKeys(uniqueKeys, source, o)
			if err != nil {
				return match{}, false, err
			}
			if matchFound {
				candidates[j] = constraintMatches // candidate found, store index (same object might match for several constraints, set ensures we only count it once)
			}
		}
	}

	if len(candidates) == 1 { // unique match found
		index := maps.Keys(candidates)[0]
		return match{
			object:  objects[index],
			matches: candidates[index],
		}, true, nil
	}

	if len(candidates) > 1 {

		if len(candidates) > 5 {
			// showing many objectIDs to a user won't actually be useful
			return match{}, false, fmt.Errorf("can't update configuration %q - unable to find exact match, %d existing objects with matching unique keys found", source.Coordinate, len(candidates))
		}

		var objectIds []string
		for i := range candidates {
			objectIds = append(objectIds, objects[i].ObjectId)
		}

		return match{}, false, fmt.Errorf("can't update configuration %q - unable to find exact match, %d existing objects with matching unique keys found: %v", source.Coordinate, len(objectIds), strings.Join(objectIds, ", "))
	}

	return match{}, false, nil // no matches found
}

type constraintMatch map[string]any

func doObjectsMatchBasedOnUniqueKeys(uniqueKeys []string, source SettingsObject, other DownloadSettingsObject) (bool, constraintMatch, error) {
	matches := make(constraintMatch)
	for _, key := range uniqueKeys {
		same, val, err := isSameValueForKey(key, source.Content, other.Value)
		if err != nil {
			return false, nil, err
		}
		if !same {
			return false, nil, nil
		}
		matches[key] = val
	}
	return true, matches, nil
}

func isSameValueForKey(key string, c1 []byte, c2 []byte) (same bool, matchingVal any, err error) {
	u := make(map[string]any)
	if err := json.Unmarshal(c1, &u); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal data for key %q: %w", key, err)
	}
	v1 := u[key]

	if err := json.Unmarshal(c2, &u); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal data for key %q: %w", key, err)
	}
	v2 := u[key]

	if cmp.Equal(v1, v2) {
		return true, v1, nil
	}
	return false, nil, nil
}

// buildPostRequestPayload builds the json that is required as body in the settings api.
// POST Request body: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings/objects/post-object#request-body-json-model
//
// To do this, we have to wrap the template in another object and send this object to the server.
// Currently, we only encode one object into an array of objects, but we can optimize it to contain multiple elements to update.
// Note payload limitations: https://www.dynatrace.com/support/help/dynatrace-api/basics/access-limit#payload-limit
func buildPostRequestPayload(ctx context.Context, obj SettingsObject, externalID string, insertAfter string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(obj.Content, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rendered config: %w", err)
	}

	data := settingsRequest{
		SchemaId:      obj.SchemaId,
		ExternalId:    externalID,
		Scope:         obj.Scope,
		Value:         value,
		SchemaVersion: obj.SchemaVersion,
		ObjectId:      obj.OriginObjectId,
		InsertAfter:   insertAfter,
	}

	// Create json obj. We currently marshal everything into an array, but we can optimize it to include multiple objects in the
	// future. Look up limits when imp
	fullObj, err := json.Marshal([]interface{}{data})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal full object: %w", err)
	}

	// compress json to require less space
	dest := bytes.Buffer{}
	if err := json.Compact(&dest, fullObj); err != nil {
		log.WithCtxFields(ctx).WithFields(field.Error(err)).Debug("Failed to compact json: %s. Using uncompressed json.\n\tJson: %v", err, string(fullObj))
		return fullObj, nil
	}

	return dest.Bytes(), nil
}

// parsePostResponse unmarshalls and parses the settings response for the post request
// The response is returned as an array for each element we send.
// Since we only send one object at the moment, we simply use the first one.
func parsePostResponse(body []byte) (DynatraceEntity, error) {

	var parsed []postResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to unmarshal response: %w. Response was: %s", err, string(body))
	}

	if len(parsed) == 0 {
		return DynatraceEntity{}, fmt.Errorf("response does not contain a single element")
	}

	if len(parsed) > 1 {
		return DynatraceEntity{}, fmt.Errorf("response does contain too many elements")
	}

	return DynatraceEntity{
		Id:   parsed[0].ObjectId,
		Name: parsed[0].ObjectId,
	}, nil
}

func (d *SettingsClient) ListSettings(ctx context.Context, schemaId string, opts ListSettingsOptions) (res []DownloadSettingsObject, err error) {
	if settings, cached := d.settingsCache.Get(schemaId); cached {
		log.WithCtxFields(ctx).Debug("Using cached settings for schema %s", schemaId)
		return filter.FilterSlice(settings, opts.Filter), nil
	}

	log.WithCtxFields(ctx).Debug("Downloading all settings for schema %s", schemaId)

	listSettingsFields := defaultListSettingsFields
	if opts.DiscardValue {
		listSettingsFields = reducedListSettingsFields
	}
	params := url.Values{
		"schemaIds": []string{schemaId},
		"pageSize":  []string{defaultPageSize},
		"fields":    []string{listSettingsFields},
	}

	result := make([]DownloadSettingsObject, 0)

	addToResult := func(body []byte) (int, error) {
		var parsed struct {
			Items []DownloadSettingsObject `json:"items"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return 0, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		result = append(result, parsed.Items...)
		return len(parsed.Items), nil
	}

	err = listPaginated(ctx, d.client, d.retrySettings.Normal, d.settingsObjectAPIPath, params, schemaId, addToResult)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings of schema %q: %w", schemaId, err)
	}

	d.settingsCache.Set(schemaId, result)

	return filter.FilterSlice(result, opts.Filter), nil
}
