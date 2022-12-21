/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"net/http"
	"net/url"
	"strings"

	. "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
)

// ConfigClient is responsible for the classic Dynatrace configs. For settings objects, the [SettingsClient] is responsible.
// Each config endpoint is described by an [Api] object to describe endpoints, structure, and behavior.
type ConfigClient interface {
	// List lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	List(a Api) (values []Value, err error)

	// ReadById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadById(a Api, id string) (json []byte, err error)

	// UpsertByName creates a given Dynatrace config if it doesn't exist and updates it otherwise using its name.
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertByName(a Api, name string, payload []byte) (entity DynatraceEntity, err error)

	// UpsertByEntityId creates or updates an existing Dynatrace entity by its id.
	// If the entity doesn't exist it is created with the according id. E.g. for alerting profiles this would be:
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... whether or not the config is already available
	UpsertByEntityId(a Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error)

	// DeleteById removes a given config for a given API using its id.
	// It calls the DELETE endpoint for the API. E.g. for alerting profiles this would be:
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteById(a Api, id string) error

	// ExistsByName checks if a config with the given name exists for the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ExistsByName(a Api, name string) (exists bool, id string, err error)
}

// KnownSettings contains externalId -> objectId
type KnownSettings map[string]string

type DownloadSettingsObject struct {
	ExternalId    string           `json:"externalId"`
	SchemaVersion string           `json:"schemaVersion"`
	SchemaId      string           `json:"schemaId"`
	ObjectId      string           `json:"objectId"`
	Scope         string           `json:"scope"`
	Value         *json.RawMessage `json:"value"`
}

// SettingsClient is the abstraction layer for CRUD operations on the Dynatrace Settings API.
// Its design is intentionally not dependent on Monaco objects.
//
// This interface exclusively accesses the [settings api] of Dynatrace.
//
// The base mechanism for all methods is the same:
// We identify objects to be updated/deleted by their external-id. If an object can not be found using its external-id, we assume
// that it does not exist.
// More documentation is written in each method's documentation.
//
// [settings api]: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings
type SettingsClient interface {
	// UpsertSettings either creates the supplied object, or updates an existing one.
	// First, we try to find the external-id of the object. If we can't find it, we create the object, if we find it, we
	// update the object.
	UpsertSettings(settings KnownSettings, obj SettingsObject) (DynatraceEntity, error) // create or update, first version only create

	// ListKnownSettings queries all settings for the given schema ID.
	// All queried objects that have an external ID will be returned.
	ListKnownSettings(schemaId string) (KnownSettings, error)

	// ListSchemas returns all schemas that the Dynatrace environment reports
	ListSchemas() (SchemaList, error)

	// ListSettings returns all settings objects for a given schema.
	ListSettings(schema string) ([]DownloadSettingsObject, error)
}

//go:generate mockgen -source=client.go -destination=client_mock.go -package=rest -imports .=github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api DynatraceClient

// DynatraceClient provides the functionality for performing basic CRUD operations on any Dynatrace API
// supported by monaco.
// It encapsulates the configuration-specific inconsistencies of certain APIs in one place to provide
// a common interface to work with. After all: A user of DynatraceClient shouldn't care about the
// implementation details of each individual Dynatrace API.
// Its design is intentionally not dependent on the Config and Environment interfaces included in monaco.
// This makes sure, that DynatraceClient can be used as a base for future tooling, which relies on
// a standardized way to access Dynatrace APIs.
type DynatraceClient interface {
	ConfigClient
	SettingsClient
}

type dynatraceClient struct {
	environmentUrl string
	token          string
	client         *http.Client
	retrySettings  retrySettings
}

var (
	_ SettingsClient  = (*dynatraceClient)(nil)
	_ ConfigClient    = (*dynatraceClient)(nil)
	_ DynatraceClient = (*dynatraceClient)(nil)
)

// NewDynatraceClient creates a new DynatraceClient
func NewDynatraceClient(environmentUrl, token string) (DynatraceClient, error) {
	return newDynatraceClient(environmentUrl, token, &http.Client{}, defaultRetrySettings)
}

func newDynatraceClient(environmentUrl, token string, client *http.Client, settings retrySettings) (*dynatraceClient, error) {
	environmentUrl = strings.TrimSuffix(environmentUrl, "/")

	if environmentUrl == "" {
		return nil, errors.New("no environment url")
	}

	if token == "" {
		return nil, errors.New("no token")
	}

	parsedUrl, err := url.ParseRequestURI(environmentUrl)
	if err != nil {
		return nil, errors.New("environment url " + environmentUrl + " was not valid")
	}

	if parsedUrl.Scheme != "https" {
		return nil, errors.New("environment url " + environmentUrl + " was not valid")
	}

	if !isNewDynatraceTokenFormat(token) {
		log.Warn("You used an old token format. Please consider switching to the new 1.205+ token format.")
		log.Warn("More information: https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication/#-dynatrace-version-1205--token-format")
	}

	return &dynatraceClient{
		environmentUrl: environmentUrl,
		token:          token,
		client:         client,
		retrySettings:  settings,
	}, nil
}

func isNewDynatraceTokenFormat(token string) bool {
	return strings.HasPrefix(token, "dt0c01.") && strings.Count(token, ".") == 2
}

func (d *dynatraceClient) UpsertSettings(settings KnownSettings, obj SettingsObject) (DynatraceEntity, error) {
	externalId := util.GenerateExternalId(obj.SchemaId, obj.Id)
	objectId, found := settings[externalId]

	if !found {
		payload, err := buildPostRequestPayload(obj, externalId)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to build settings object for upsert: %w", err)
		}

		requestUrl := d.environmentUrl + pathSettingsObjects

		resp, err := post(d.client, requestUrl, payload, d.token)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to upsert dynatrace obj: %w", err)
		}

		if !success(resp) {
			return DynatraceEntity{}, fmt.Errorf("failed to update settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
		}

		entity, err := parsePostResponse(resp)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
		}

		log.Debug("\tCreated object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
		return entity, nil
	} else {
		payload, err := buildPutRequestPayload(obj, externalId)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to build settings object for upsert: %w", err)
		}

		requestUrl := d.environmentUrl + pathSettingsObjects + "/" + objectId

		resp, err := put(d.client, requestUrl, payload, d.token)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to upsert dynatrace obj: %w", err)
		}

		if !success(resp) {
			return DynatraceEntity{}, fmt.Errorf("failed to update settings object with externalId %s (HTTP %d)!\n\tResponse was: %s", externalId, resp.StatusCode, string(resp.Body))
		}

		entity, err := parsePutResponse(resp)
		if err != nil {
			return DynatraceEntity{}, fmt.Errorf("failed to parse response: %w", err)
		}

		log.Debug("\tUpdated object %s (%s) with externalId %s", obj.Id, obj.SchemaId, externalId)
		return entity, nil
	}
}

func (d *dynatraceClient) List(api Api) (values []Value, err error) {

	fullUrl := api.GetUrl(d.environmentUrl)
	values, err = getExistingValuesFromEndpoint(d.client, api, fullUrl, d.token, d.retrySettings)
	return values, err
}

func (d *dynatraceClient) ReadById(api Api, id string) (json []byte, err error) {
	var dtUrl string
	isSingleConfigurationApi := api.IsSingleConfigurationApi()

	if isSingleConfigurationApi {
		dtUrl = api.GetUrl(d.environmentUrl)
	} else {
		dtUrl = api.GetUrl(d.environmentUrl) + "/" + url.PathEscape(id)
	}

	response, err := get(d.client, dtUrl, d.token)

	if err != nil {
		return nil, err
	}

	if !success(response) {
		return nil, fmt.Errorf("Failed to get existing config for api %v (HTTP %v)!\n    Response was: %v", api.GetId(), response.StatusCode, string(response.Body))
	}

	return response.Body, nil
}

func (d *dynatraceClient) DeleteById(api Api, id string) error {

	return deleteConfig(d.client, api.GetUrl(d.environmentUrl), d.token, id)
}

func (d *dynatraceClient) ExistsByName(api Api, name string) (exists bool, id string, err error) {
	url := api.GetUrl(d.environmentUrl)

	existingObjectId, err := getObjectIdIfAlreadyExists(d.client, api, url, name, d.token, d.retrySettings)
	return existingObjectId != "", existingObjectId, err
}

func (d *dynatraceClient) UpsertByName(api Api, name string, payload []byte) (entity DynatraceEntity, err error) {

	if api.GetId() == "extension" {
		fullUrl := api.GetUrl(d.environmentUrl)
		return uploadExtension(d.client, fullUrl, name, payload, d.token)
	}
	return upsertDynatraceObject(d.client, d.environmentUrl, name, api, payload, d.token, d.retrySettings)
}

func (d *dynatraceClient) UpsertByEntityId(api Api, entityId string, name string, payload []byte) (entity DynatraceEntity, err error) {
	return upsertDynatraceEntityById(d.client, d.environmentUrl, entityId, name, api, payload, d.token, d.retrySettings)
}

type listEntry struct {
	ObjectId   string `json:"objectId"`
	ExternalId string `json:"externalId"`
}

type listResponse struct {
	Items []listEntry `json:"items"`
}

func (d *dynatraceClient) ListKnownSettings(schemaId string) (KnownSettings, error) {

	u, err := url.Parse(d.environmentUrl + pathSettingsObjects)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url '%s': %w", d.environmentUrl+pathSettingsObjects, err)
	}

	// TODO: This will fail if any schema is unknown - has to be split up into multiple calls for each schema
	params := url.Values{
		"schemaIds": []string{schemaId},
		"pageSize":  []string{"500"},
		"fields":    []string{"externalId,objectId"},
	}
	u.RawQuery = params.Encode()

	resp, err := getWithRetry(d.client, u.String(), d.token, d.retrySettings.normal)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	if !success(resp) {
		return nil, fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	result := make(KnownSettings)
	for {
		var parsed listResponse
		if err := json.Unmarshal(resp.Body, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		for _, v := range parsed.Items {
			if v.ExternalId != "" {
				result[v.ExternalId] = v.ObjectId
			}
		}

		if resp.NextPageKey != "" {
			u = addNextPageQueryParams(u, resp.NextPageKey)

			resp, err = getWithRetry(d.client, u.String(), d.token, d.retrySettings.normal)

			if err != nil {
				return nil, err
			}

			if !success(resp) {
				return nil, fmt.Errorf("Failed to get further configs from Settings API (HTTP %d)!\n    Response was: %s", resp.StatusCode, string(resp.Body))
			}

		} else {
			break
		}
	}

	return result, nil
}

type SchemaListResponse struct {
	Items SchemaList `json:"items"`
}
type SchemaList []struct {
	SchemaId string `json:"schemaId"`
}

func (d *dynatraceClient) ListSchemas() (SchemaList, error) {
	u, err := url.Parse(d.environmentUrl + pathSchemas)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	// getting all schemas does not have pagination
	resp, err := get(d.client, u.String(), d.token)
	if err != nil {
		return nil, fmt.Errorf("failed to GET schemas: %w", err)
	}

	if !success(resp) {
		return nil, fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	var result SchemaListResponse
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result.Items, nil
}

func (d *dynatraceClient) ListSettings(schema string) ([]DownloadSettingsObject, error) {

	u, err := url.Parse(d.environmentUrl + pathSettingsObjects)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse url '%s': %w", d.environmentUrl+pathSettingsObjects, err)
	}

	params := url.Values{
		"schemaIds": []string{schema},
		"pageSize":  []string{"500"},
		"fields":    []string{"objectId,value,externalId,schemaVersion,schemaId,scope"},
	}
	u.RawQuery = params.Encode()

	resp, err := getWithRetry(d.client, u.String(), d.token, d.retrySettings.normal)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	if !success(resp) {
		return nil, fmt.Errorf("request failed with HTTP (%d).\n\tResponse content: %s", resp.StatusCode, string(resp.Body))
	}

	result := make([]DownloadSettingsObject, 0)
	for {
		var parsed struct {
			Items []DownloadSettingsObject `json:"items"`
		}
		if err := json.Unmarshal(resp.Body, &parsed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		result = append(result, parsed.Items...)

		if resp.NextPageKey != "" {
			u = addNextPageQueryParams(u, resp.NextPageKey)

			resp, err = getWithRetry(d.client, u.String(), d.token, d.retrySettings.normal)

			if err != nil {
				return nil, err
			}

			if !success(resp) {
				return nil, fmt.Errorf("Failed to get further configs from Settings API (HTTP %d)!\n    Response was: %s", resp.StatusCode, string(resp.Body))
			}

		} else {
			break
		}
	}

	return result, nil
}
