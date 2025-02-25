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
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/filter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log/field"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/version"
	dtVersion "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/version"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

// SettingsResourceContext.Operations possibilities
const (
	DeleteOperation = "delete"
	WriteOperation  = "write"
)

// DownloadSettingsObject is the response type for the ListSettings operation
type DownloadSettingsObject struct {
	ExternalId    string          `json:"externalId"`
	SchemaVersion string          `json:"schemaVersion"`
	SchemaId      string          `json:"schemaId"`
	ObjectId      string          `json:"objectId"`
	Scope         string          `json:"scope"`
	Value         json.RawMessage `json:"value"`
	//Deprecated in the API used only as fallback replaced by ResourceContext
	ModificationInfo *SettingsModificationInfo `json:"modificationInfo"`
	ResourceContext  *SettingsResourceContext  `json:"resourceContext"`
}

func (settingObject *DownloadSettingsObject) IsDeletable() bool {
	if settingObject.ResourceContext != nil {
		return slices.Contains(settingObject.ResourceContext.Operations, DeleteOperation)
	}

	if settingObject.ModificationInfo != nil {
		return settingObject.ModificationInfo.Deletable
	}

	return true
}

func (settingObject *DownloadSettingsObject) IsModifiable() bool {
	if settingObject.ResourceContext != nil {
		return slices.Contains(settingObject.ResourceContext.Operations, WriteOperation)
	}

	if settingObject.ModificationInfo != nil {
		return settingObject.ModificationInfo.Modifiable
	}

	return true
}

func (settingObject *DownloadSettingsObject) IsMovable() bool {
	if settingObject.ResourceContext != nil {
		//API allows the parameter to be optional, so more logic is needed to handle it
		if settingObject.ResourceContext.Movable != nil {
			return *settingObject.ResourceContext.Movable
		}
		return true
	}

	if settingObject.ModificationInfo != nil {
		return settingObject.ModificationInfo.Movable
	}

	return true
}

func (settingObject *DownloadSettingsObject) GetModifiablePaths() []string {
	if settingObject.ResourceContext != nil {
		return settingObject.ResourceContext.ModifiablePaths
	}

	return settingObject.ModificationInfo.ModifiablePaths
}

type SettingsModificationInfo struct {
	Deletable       bool     `json:"deletable"`
	Modifiable      bool     `json:"modifiable"`
	Movable         bool     `json:"movable"`
	ModifiablePaths []string `json:"modifiablePaths"`
}

type SettingsResourceContext struct {
	Operations      []string `json:"operations"`
	Movable         *bool    `json:"modifications:movable"`
	ModifiablePaths []string `json:"modifications:modifiablePaths"`
}

type UpsertSettingsOptions struct {
	OverrideRetry *RetrySetting

	// InsertAfter is the position at where the object should be inserted.
	// It can be an arbitrary ID of another settings object.
	// It must be nil if the schema is an unordered schema. If it's not set for ordered schemas, it is handled like InsertPositionBack.
	//
	// It supports the following magic values to insert at the FRONT and BOTTOM of the list:
	//   - FRONT (InsertPositionFront) -> insert at the front of the list
	//   - BACK (InsertPositionBack) -> insert at the back of the list
	InsertAfter *string
}

const (
	InsertPositionFront = "FRONT"
	InsertPositionBack  = "BACK"
)

// defaultListSettingsFields  are the fields we are interested in when getting setting objects
const defaultListSettingsFields = "objectId,value,externalId,schemaVersion,schemaId,scope,modificationInfo"

// reducedListSettingsFields are the fields we are interested in when getting settings objects but don't care about the
// actual value payload
const reducedListSettingsFields = "objectId,externalId,schemaVersion,schemaId,scope,modificationInfo"
const defaultPageSize = "500"

// ListSettingsOptions are additional options for the ListSettings method
// of the Settings client
type ListSettingsOptions struct {
	// DiscardValue specifies whether the value field of the returned
	// settings object shall be included in the payload
	DiscardValue bool
	// ListSettingsFilter can be set to pre-filter the result given a special logic
	Filter ListSettingsFilter
}

// ListSettingsFilter can be used to filter fetched settings objects with custom criteria, e.g. o.ExternalId == ""
type ListSettingsFilter func(DownloadSettingsObject) bool

type SettingsClient struct {
	// serverVersion is the Dynatrace server version the
	// client will be interacting with
	serverVersion version.Version

	// client is a rest client used to target platform enabled environments
	client *corerest.Client

	// settingsSchemaAPIPath is the API path to use for accessing settings schemas
	settingsSchemaAPIPath string

	//  settingsObjectAPIPath is the API path to use for accessing settings objects
	settingsObjectAPIPath string

	// retrySettings are the settings to be used for retrying failed http requests
	retrySettings RetrySettings

	// generateExternalID is used to generate an external id for settings 2.0 objects
	generateExternalID idutils.ExternalIDGenerator

	// settingsCache caches settings objects
	settingsCache cache.Cache[[]DownloadSettingsObject]

	// schemaCache caches schema constraints
	schemaCache cache.Cache[Schema]
}

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
		SchemaId      string  `json:"schemaId"`
		ExternalId    string  `json:"externalId,omitempty"`
		Scope         string  `json:"scope"`
		Value         any     `json:"value"`
		SchemaVersion string  `json:"schemaVersion,omitempty"`
		ObjectId      string  `json:"objectId,omitempty"`
		InsertAfter   *string `json:"insertAfter,omitempty"`
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

const (
	settingsSchemaAPIPathClassic  = "/api/v2/settings/schemas"
	settingsSchemaAPIPathPlatform = "/platform/classic/environment-api/v2/settings/schemas"
	settingsObjectAPIPathClassic  = "/api/v2/settings/objects"
	settingsObjectAPIPathPlatform = "/platform/classic/environment-api/v2/settings/objects"
)

func WithExternalIDGenerator(g idutils.ExternalIDGenerator) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		d.generateExternalID = g
	}
}

// WithRetrySettings sets the retry settings to be used by the DynatraceClient
func WithRetrySettings(retrySettings RetrySettings) func(*SettingsClient) {
	return func(d *SettingsClient) {
		d.retrySettings = retrySettings
	}
}

// WithServerVersion sets the Dynatrace version of the Dynatrace server/tenant the client will be interacting with
func WithServerVersion(serverVersion version.Version) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		d.serverVersion = serverVersion
	}
}

// WithAutoServerVersion can be used to let the client automatically determine the Dynatrace server version
// during creation using newDynatraceClient. If the server version is already known WithServerVersion should be used.
// Do not use this with NewPlatformSettingsClient() as the client will not work and cause an error to be logged.
func WithAutoServerVersion(ctx context.Context) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		var serverVersion version.Version
		var err error

		d.serverVersion = version.UnknownVersion
		if d.client == nil {
			return
		}

		serverVersion, err = dtVersion.GetDynatraceVersion(ctx, d.client) //this will send the default user-agent
		if err != nil {
			log.WithFields(field.Error(err)).Warn("Unable to determine Dynatrace server version: %v", err)
			return
		}
		d.serverVersion = serverVersion
	}
}

// WithCachingDisabled allows disabling the client's builtin caching mechanism for schema constraints and settings objects.
// Disabling the caching is recommended in situations where configs are fetched immediately after their creation (e.g. in test scenarios).
func WithCachingDisabled(disabled bool) func(client *SettingsClient) {
	return func(d *SettingsClient) {
		if disabled {
			d.schemaCache = &cache.NoopCache[Schema]{}
			d.settingsCache = &cache.NoopCache[[]DownloadSettingsObject]{}
		}
	}
}

// NewPlatformSettingsClient creates a new settings client to be used for platform enabled environments
//
//nolint:dupl
func NewPlatformSettingsClient(client *corerest.Client, opts ...func(dynatraceClient *SettingsClient)) (*SettingsClient, error) {
	d := &SettingsClient{
		serverVersion:         version.Version{},
		client:                client,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathPlatform,
		settingsObjectAPIPath: settingsObjectAPIPathPlatform,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

// NewClassicSettingsClient creates a new settings client to be used for classic environments.
//
//nolint:dupl
func NewClassicSettingsClient(client *corerest.Client, opts ...func(dynatraceClient *SettingsClient)) (*SettingsClient, error) {
	d := &SettingsClient{
		serverVersion:         version.Version{},
		client:                client,
		retrySettings:         DefaultRetrySettings,
		settingsSchemaAPIPath: settingsSchemaAPIPathClassic,
		settingsObjectAPIPath: settingsObjectAPIPathClassic,
		generateExternalID:    idutils.GenerateExternalIDForSettingsObject,
		settingsCache:         &cache.DefaultCache[[]DownloadSettingsObject]{},
		schemaCache:           &cache.DefaultCache[Schema]{},
	}

	for _, o := range opts {
		if o != nil {
			o(d)
		}
	}
	return d, nil
}

func (d *SettingsClient) Cache(ctx context.Context, schemaID string) error {
	_, err := d.List(ctx, schemaID, ListSettingsOptions{})
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

func (d *SettingsClient) GetSchema(ctx context.Context, schemaID string) (constraints Schema, err error) {
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

	fetchedSettingObj, err := d.Get(ctx, obj.OriginObjectId)
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

// Upsert creates or updates remote settings objects.
// The logic to find the correct object to update is as follows:
//  1. We try to match the unique-constrains of the object
//  2. We try to find the correct object by checking the legacy-external-id, the external-id, as well as the given originObjectId
//
// If we find an object, we update it. If we don't, a new one will be created.
//
// Note: If the Dynatrace version of the remote system is <262, nothing will be performed and an error is returned.
func (d *SettingsClient) Upsert(ctx context.Context, obj SettingsObject, upsertOptions UpsertSettingsOptions) (result DynatraceEntity, err error) {
	if !d.serverVersion.Invalid() && d.serverVersion.SmallerThan(version.Version{Major: 1, Minor: 262, Patch: 0}) {
		return d.handleUpsertUnsupportedVersion(ctx, obj)
	}

	// The objectID of the object we want to update
	remoteObjectId := ""

	if matchingObject, found, err := d.findObjectWithMatchingConstraints(ctx, obj); err != nil {
		return DynatraceEntity{}, err
	} else if found {

		var props []string
		for k, v := range matchingObject.matches {
			props = append(props, fmt.Sprintf("(%v = %v)", k, v))
		}

		log.WithCtxFields(ctx).Debug("Updating existing object %q with matching unique properties: %v", matchingObject.object.ObjectId, strings.Join(props, ", "))
		remoteObjectId = matchingObject.object.ObjectId
	}

	// generate legacy external ID without project name.
	// and check if settings object with that external ID exists
	// This exists for avoiding breaking changes when we enhanced external id generation with full coordinates (incl. project name)
	// This can be removed in a later release of monaco
	legacyExternalID, err := d.generateExternalID(coordinate.Coordinate{Type: obj.Coordinate.Type, ConfigId: obj.Coordinate.ConfigId})
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("unable to generate external id: %w", err)
	}

	settingsWithExternalID, err := d.List(ctx, obj.SchemaId, ListSettingsOptions{
		Filter: func(object DownloadSettingsObject) bool { return object.ExternalId == legacyExternalID },
	})
	if err != nil {
		return DynatraceEntity{}, err
	}

	if len(settingsWithExternalID) > 0 {
		remoteObjectId = settingsWithExternalID[0].ObjectId
	}

	externalID, err := d.generateExternalID(obj.Coordinate)
	if err != nil {
		return DynatraceEntity{}, fmt.Errorf("unable to generate external id: %w", err)
	}

	// If the server contains two configs, one with the origin-object-id and a second config with the externalID,
	// it is not possible to update the setting using the externalId and origin-object-id on the same POST request,
	// as two settings objects can be the target of the change. In this case, we remove the origin-object-id
	// and only update the object using the externalId.
	settings, err := d.List(ctx, obj.SchemaId, ListSettingsOptions{
		Filter: func(object DownloadSettingsObject) bool {
			return object.ObjectId == obj.OriginObjectId || object.ExternalId == externalID
		},
	})
	if err != nil {
		return DynatraceEntity{}, err
	}
	if len(settings) == 1 {
		remoteObjectId = settings[0].ObjectId
	} else if len(settings) == 2 {
		var exIdSetting, ooIdSetting string
		if settings[0].ExternalId == externalID {
			exIdSetting = settings[0].ObjectId
			ooIdSetting = settings[1].ObjectId
		} else {
			exIdSetting = settings[1].ObjectId
			ooIdSetting = settings[0].ObjectId
		}

		log.WithCtxFields(ctx).Warn("Found two configs, one with the defined originObjectId (%q), and one with the expected monaco externalId (%q). Updating the one with the expected externalId.", ooIdSetting, exIdSetting)
		remoteObjectId = ""
	}

	if schema, ok := d.schemaCache.Get(obj.SchemaId); ok {
		if upsertOptions.InsertAfter != nil && !schema.Ordered {
			return DynatraceEntity{}, fmt.Errorf("'%s' is not an ordered setting, hence 'insertAfter' is not supported for this type of setting object", obj.SchemaId)
		}
	}

	payload, err := buildPostRequestPayload(ctx, remoteObjectId, obj, externalID, upsertOptions.InsertAfter)
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
	constraints, err := d.GetSchema(ctx, source.SchemaId)
	if err != nil {
		return match{}, false, fmt.Errorf("unable to get details for schema %q: %w", source.SchemaId, err)
	}

	if len(constraints.UniqueProperties) == 0 {
		return match{}, false, nil
	}

	objects, err := d.List(ctx, source.SchemaId, ListSettingsOptions{})
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

func isSameValueForKey(targetPath string, c1 []byte, c2 []byte) (same bool, matchingVal any, err error) {
	unmarshalledSourceConfig := make(map[string]any)
	if err := json.Unmarshal(c1, &unmarshalledSourceConfig); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal data for key %q: %w", targetPath, err)
	}

	keys := explodePath(targetPath)
	value1 := recursiveSearch(unmarshalledSourceConfig, keys)

	unmarshalledObjectConfig := make(map[string]any)
	if err := json.Unmarshal(c2, &unmarshalledObjectConfig); err != nil {
		return false, nil, fmt.Errorf("failed to unmarshal data for key %q: %w", targetPath, err)
	}

	value2 := recursiveSearch(unmarshalledObjectConfig, keys)

	// The nil check here is to prevent constraint field that is not in the payload to match(nil==nil)
	if value1 != nil && value2 != nil && cmp.Equal(value1, value2) {
		return true, value1, nil
	}
	return false, nil, nil
}

// Recursive search allows for nil values in case a field is not in the payload
func recursiveSearch(nestedMap map[string]any, keys []string) any {
	currentMap := nestedMap
	value, found := currentMap[keys[0]]
	if found {
		if nestedMap, ok := value.(map[string]interface{}); ok && len(keys) > 1 {
			return recursiveSearch(nestedMap, keys[1:])
		}
		return value
	}

	return nil
}

// explodePath splits targetPath by "/", this is the format of settings api.
// If no "/" is present the string is returned as is. If in future there should be other separators expand logic here.
func explodePath(targetPath string) []string {
	return strings.Split(targetPath, "/")
}

// buildPostRequestPayload builds the json that is required as body in the settings api.
// POST Request body: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings/objects/post-object#request-body-json-model
//
// To do this, we have to wrap the template in another object and send this object to the server.
// Currently, we only encode one object into an array of objects, but we can optimize it to contain multiple elements to update.
// Note payload limitations: https://www.dynatrace.com/support/help/dynatrace-api/basics/access-limit#payload-limit
func buildPostRequestPayload(ctx context.Context, remoteObjectId string, obj SettingsObject, externalID string, insertAfter *string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(obj.Content, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rendered config: %w", err)
	}

	insertPosition := insertAfterToPayloadValue(insertAfter)

	data := settingsRequest{
		SchemaId:      obj.SchemaId,
		ExternalId:    externalID,
		Scope:         obj.Scope,
		Value:         value,
		SchemaVersion: obj.SchemaVersion,
		ObjectId:      remoteObjectId,
		InsertAfter:   insertPosition,
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

// insertAfterToPayloadValue converts the insertAfter value to the propler
// value required in the payload.
//
// For POST (only method that we use), it must be as follows:
//
//   - insert to the front: `insertAfter` to ”
//   - insert to the back: `insertAfter` to nil (default behavior of monaco)
//   - insert after another item: `insertAfter` to the ID of the item
func insertAfterToPayloadValue(insertAfter *string) *string {

	if insertAfter == nil {
		return nil
	}

	switch *insertAfter {
	case InsertPositionBack:
		return nil
	case InsertPositionFront:
		var empty = ""
		return &empty
	default:
		return insertAfter
	}
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

func (d *SettingsClient) List(ctx context.Context, schemaId string, opts ListSettingsOptions) (res []DownloadSettingsObject, err error) {
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

func (d *SettingsClient) Get(ctx context.Context, objectId string) (res *DownloadSettingsObject, err error) {
	resp, err := coreapi.AsResponseOrError(d.client.GET(ctx, d.settingsObjectAPIPath+"/"+objectId, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		return nil, err
	}

	var result DownloadSettingsObject
	if err = json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal settings object: %w", err)
	}

	return &result, nil
}

func (d *SettingsClient) Delete(ctx context.Context, objectID string) error {
	_, err := coreapi.AsResponseOrError(d.client.DELETE(ctx, d.settingsObjectAPIPath+"/"+objectID, corerest.RequestOptions{CustomShouldRetryFunc: corerest.RetryIfTooManyRequests}))
	if err != nil {
		apiError := coreapi.APIError{}
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
			log.Debug("No settings object with id '%s' found to delete (HTTP 404 response)", objectID)
			return nil
		}
		return err
	}

	return nil
}
