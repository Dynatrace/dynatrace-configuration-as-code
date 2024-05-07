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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/filter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"
	"net/http"
	"net/url"
	"strings"
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

	SchemaConstraints struct {
		SchemaId         string
		Ordered          bool
		UniqueProperties [][]string
	}

	SchemaList []struct {
		SchemaId string `json:"schemaId"`
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

	// schemaDetailsResponse is the response type returned by the fetchSchemasConstraints operation
	schemaDetailsResponse struct {
		SchemaId          string             `json:"schemaId"`
		Ordered           bool               `json:"ordered"`
		SchemaConstraints []schemaConstraint `json:"schemaConstraints"`
	}
)

func (d *DynatraceClient) ListSchemas() (schemas SchemaList, err error) {
	d.limiter.ExecuteBlocking(func() {
		schemas, err = d.listSchemas(context.TODO())
	})
	return
}

func (d *DynatraceClient) listSchemas(ctx context.Context) (SchemaList, error) {
	u, err := url.Parse(d.environmentURL + d.settingsSchemaAPIPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// getting all schemas does not have pagination
	resp, err := d.platformClient.Get(ctx, u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to GET schemas: %w", err)
	}

	if !resp.IsSuccess() {
		return nil, rest.NewRespErr(fmt.Sprintf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodGet, u.String())
	}

	var result SchemaListResponse
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, rest.NewRespErr("failed to unmarshal response", resp).WithRequestInfo(http.MethodGet, u.String()).WithErr(err)
	}

	if result.TotalCount != len(result.Items) {
		log.Warn("Total count of settings 2.0 schemas (=%d) does not match with count of actually downloaded settings 2.0 schemas (=%d)", result.TotalCount, len(result.Items))
	}

	return result.Items, nil
}

func (d *DynatraceClient) FetchSchemasConstraints(schemaID string) (constraints SchemaConstraints, err error) {
	d.limiter.ExecuteBlocking(func() {
		constraints, err = d.fetchSchemasConstraints(context.TODO(), schemaID)
	})
	return
}

func (d *DynatraceClient) fetchSchemasConstraints(ctx context.Context, schemaID string) (SchemaConstraints, error) {
	if ret, cached := d.schemaConstraintsCache.Get(schemaID); cached {
		return ret, nil
	}

	ret := SchemaConstraints{SchemaId: schemaID}
	u, err := url.JoinPath(d.environmentURL, d.settingsSchemaAPIPath, schemaID)
	if err != nil {
		return SchemaConstraints{}, fmt.Errorf("failed to parse url: %w", err)
	}

	r, err := d.platformClient.Get(ctx, u)
	if err != nil {
		return SchemaConstraints{}, fmt.Errorf("failed to GET schema details for %q: %w", schemaID, err)
	}

	var sd schemaDetailsResponse
	err = json.Unmarshal(r.Body, &sd)
	if err != nil {
		return SchemaConstraints{}, rest.NewRespErr("failed to unmarshal response", r).WithRequestInfo(http.MethodGet, u).WithErr(err)
	}

	for _, sc := range sd.SchemaConstraints {
		if sc.Type == "UNIQUE" {
			ret.UniqueProperties = append(ret.UniqueProperties, sc.UniqueProperties)
		}
	}
	ret.Ordered = sd.Ordered

	d.schemaConstraintsCache.Set(schemaID, ret)
	return ret, nil
}

func (d *DynatraceClient) UpsertSettings(ctx context.Context, obj SettingsObject, options UpsertSettingsOptions) (result DynatraceEntity, err error) {
	d.limiter.ExecuteBlocking(func() {
		result, err = d.upsertSettings(ctx, obj, options)
	})
	return
}

func (d *DynatraceClient) upsertSettings(ctx context.Context, obj SettingsObject, options UpsertSettingsOptions) (DynatraceEntity, error) {
	// special handling for updating settings 2.0 objects on tenants with version pre 1.262.0
	// Tenants with versions < 1.262 are not able to handle updates of existing
	// settings 2.0 objects that are non-deletable.
	// So we check if the object with originObjectID already exists, if yes and the tenant is older than 1.262
	// then we cannot perform the upsert operation
	if !d.serverVersion.Invalid() && d.serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262, Patch: 0}) {
		fetchedSettingObj, err := d.getSettingById(ctx, obj.OriginObjectId)
		if err != nil && !errors.Is(err, ErrSettingNotFound) {
			return DynatraceEntity{}, fmt.Errorf("unable to fetch settings object with object id %q: %w", obj.OriginObjectId, err)
		}
		if fetchedSettingObj != nil {
			log.WithCtxFields(ctx).Warn("Unable to update Settings 2.0 object of schema %q and object id %q on Dynatrace environment with a version < 1.262.0", obj.SchemaId, obj.OriginObjectId)
			return DynatraceEntity{
				Id:   fetchedSettingObj.ObjectId,
				Name: fetchedSettingObj.ObjectId,
			}, nil
		}
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

	settingsWithExternalID, err := d.listSettings(ctx, obj.SchemaId, ListSettingsOptions{
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
	settings, err := d.listSettings(ctx, obj.SchemaId, ListSettingsOptions{
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

	payload, err := buildPostRequestPayload(ctx, obj, externalID, options.InsertAfter)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to build settings object: %w", err)
	}

	var retrySetting rest.RetrySetting
	if options.OverrideRetry != nil {
		retrySetting = *options.OverrideRetry
	} else {
		retrySetting = d.retrySettings.Normal
	}

	requestUrl := d.environmentURL + d.settingsObjectAPIPath
	resp, err := rest.SendWithRetryWithInitialTry(ctx, d.platformClient.Post, requestUrl, payload, retrySetting)
	if err != nil {
		d.settingsCache.Delete(obj.SchemaId)
		return DynatraceEntity{}, fmt.Errorf("failed to create or update Settings object with externalId %s: %w", externalID, err)
	}

	if !resp.IsSuccess() {
		d.settingsCache.Delete(obj.SchemaId)
		return DynatraceEntity{}, rest.NewRespErr(fmt.Sprintf("failed to create or update Settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalID, resp.StatusCode, string(resp.Body)), resp).WithRequestInfo(http.MethodPost, requestUrl)
	}

	entity, err := parsePostResponse(resp)
	if err != nil {
		return DynatraceEntity{}, rest.NewRespErr("failed to parse response", resp).WithRequestInfo(http.MethodPost, requestUrl).WithErr(err)
	}

	log.WithCtxFields(ctx).Debug("Created/Updated object %s (%s) with externalId %s", obj.Coordinate.ConfigId, obj.SchemaId, externalID)
	return entity, nil
}

type match struct {
	object  DownloadSettingsObject
	matches constraintMatch
}

func (d *DynatraceClient) findObjectWithMatchingConstraints(ctx context.Context, source SettingsObject) (match, bool, error) {
	constraints, err := d.fetchSchemasConstraints(ctx, source.SchemaId)
	if err != nil {
		return match{}, false, fmt.Errorf("unable to get details for schema %q: %w", source.SchemaId, err)
	}

	if len(constraints.UniqueProperties) == 0 {
		return match{}, false, nil
	}

	objects, err := d.listSettings(ctx, source.SchemaId, ListSettingsOptions{})
	if err != nil {
		return match{}, false, fmt.Errorf("unable to get existing settings objects for %q schema: %w", source.SchemaId, err)
	}

	target, found, err := findObjectWithSameConstraints(constraints, source, objects)
	if err != nil {
		return match{}, false, err
	}
	return target, found, nil
}

func findObjectWithSameConstraints(schema SchemaConstraints, source SettingsObject, objects []DownloadSettingsObject) (match, bool, error) {
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
func parsePostResponse(resp rest.Response) (DynatraceEntity, error) {

	var parsed []postResponse
	if err := json.Unmarshal(resp.Body, &parsed); err != nil {
		return DynatraceEntity{}, fmt.Errorf("failed to unmarshal response: %w. Response was: %s", err, string(resp.Body))
	}

	if len(parsed) == 0 {
		return DynatraceEntity{}, fmt.Errorf("response did not contain a single element")
	}

	if len(parsed) > 1 {
		return DynatraceEntity{}, fmt.Errorf("response did contain too many elements")
	}

	return DynatraceEntity{
		Id:   parsed[0].ObjectId,
		Name: parsed[0].ObjectId,
	}, nil
}

func (d *DynatraceClient) ListSettings(ctx context.Context, schemaId string, opts ListSettingsOptions) (res []DownloadSettingsObject, err error) {
	d.limiter.ExecuteBlocking(func() {
		res, err = d.listSettings(ctx, schemaId, opts)
	})
	return
}

func (d *DynatraceClient) listSettings(ctx context.Context, schemaId string, opts ListSettingsOptions) ([]DownloadSettingsObject, error) {

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

	u, err := buildUrl(d.environmentURL, d.settingsObjectAPIPath, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for schema %q: %w", schemaId, err)
	}

	_, err = rest.ListPaginated(ctx, d.platformClient, d.retrySettings, u, schemaId, addToResult)
	if err != nil {
		return nil, fmt.Errorf("failed to list settings of schema %q: %w", schemaId, err)
	}

	d.settingsCache.Set(schemaId, result)

	return filter.FilterSlice(result, opts.Filter), nil
}
