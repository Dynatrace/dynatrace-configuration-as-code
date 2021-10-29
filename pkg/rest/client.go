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
	"net/http"
	"net/url"
	"strings"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"

	. "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
)

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

	// List lists the available configs for an API.
	// It calls the underlying GET endpoint of the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	// The result is expressed using a list of Value (id and name tuples).
	List(a Api) (values []Value, err error)

	// ReadByName reads a Dynatrace config identified by name from the given API.
	// It calls the underlying GET endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to get the id of the existing alerting profile
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadByName(a Api, name string) (json []byte, err error)

	// ReadById reads a Dynatrace config identified by id from the given API.
	// It calls the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles/<id> ... to get the alerting profile
	ReadById(a Api, name string) (json []byte, err error)

	// UpsertByName creates a given Dynatrace config if it doesn't exists and updates it otherwise using its name
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	UpsertByName(a Api, name string, payload []byte) (entity DynatraceEntity, err error)

	// ValidateByName validates a given Dynatrace settings 2.0 config
	// It calls the underlying GET, POST, and PUT endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to check if the config is already available
	//    POST <environment-url>/api/config/v1/alertingProfiles ... afterwards, if the config is not yet available
	//    PUT <environment-url>/api/config/v1/alertingProfiles/<id> ... instead of POST, if the config is already available
	// ATTENTION: This only works for settings 2.0 APIs
	ValidateByName(a Api, name string, payload []byte) (valid bool, err error)

	// DeleteByName removes a given config for a given API using its name.
	// It calls the underlying GET and DELETE endpoints for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles ... to get the id of the existing config
	//    DELETE <environment-url>/api/config/v1/alertingProfiles/<id> ... to delete the config
	DeleteByName(a Api, name string) error

	// ExistsByName checks if a config with the given name exists for the given API.
	// It cally the underlying GET endpoint for the API. E.g. for alerting profiles this would be:
	//    GET <environment-url>/api/config/v1/alertingProfiles
	ExistsByName(a Api, name string) (exists bool, id string, err error)
}

type dynatraceClientImpl struct {
	environmentUrl    string
	token             string
	client            *http.Client
	settings20Schemas map[string]Settings20SchemaItemResponse
}

// NewDynatraceClient creates a new DynatraceClient
func NewDynatraceClient(environmentUrl, token string) (DynatraceClient, error) {

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
		util.Log.Warn("You used an old token format. Please consider switching to the new 1.205+ token format.")
		util.Log.Warn("More information: https://www.dynatrace.com/support/help/dynatrace-api/basics/dynatrace-api-authentication/#-dynatrace-version-1205--token-format")
	}

	client := &http.Client{}

	// Get settings 2.0 schemas:
	resp, err := get(client, Settings20SchemaApi.GetUrlFromEnvironmentUrl(environmentUrl), token)
	if err != nil {
		return nil, err
	}

	// Unmarshal the schemas from the API:
	var existingSchemas Settings20SchemaResponse
	err = json.Unmarshal(resp.Body, &existingSchemas)
	if util.CheckError(err, "Cannot unmarshal API response for existing schemas") {
		return nil, err
	}

	// Build a map (for handy access)
	schemaMapOfEnvironment := make(map[string]Settings20SchemaItemResponse)
	for _, schema := range existingSchemas.Items {
		schemaMapOfEnvironment[schema.SchemaId] = schema
	}

	return &dynatraceClientImpl{
		environmentUrl:    environmentUrl,
		token:             token,
		client:            &http.Client{},
		settings20Schemas: schemaMapOfEnvironment,
	}, nil
}

func isNewDynatraceTokenFormat(token string) bool {
	return strings.HasPrefix(token, "dt0c01.") && strings.Count(token, ".") == 2
}

func (d *dynatraceClientImpl) List(api Api) (values []Value, err error) {

	fullUrl := api.GetUrlFromEnvironmentUrl(d.environmentUrl)
	uploader := newConfigUploader(api)
	values, err = uploader.getExistingValuesFromEndpoint(d, api, fullUrl)
	return values, err
}

func (d *dynatraceClientImpl) ReadByName(api Api, name string) (json []byte, err error) {

	exists, id, err := d.ExistsByName(api, name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New("404 - no config found with name " + name)
	}

	return d.ReadById(api, id)
}

func (d *dynatraceClientImpl) ReadById(api Api, id string) (json []byte, err error) {
	fullUrl := api.GetUrlFromEnvironmentUrl(d.environmentUrl) + "/" + id
	response, err := get(d.client, fullUrl, d.token)

	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

func (d *dynatraceClientImpl) DeleteByName(api Api, name string) error {

	uploader := newConfigUploader(api)
	return uploader.deleteDynatraceObject(d, api, name, api.GetUrlFromEnvironmentUrl(d.environmentUrl))
}

func (d *dynatraceClientImpl) ExistsByName(api Api, name string) (exists bool, id string, err error) {

	uploader := newConfigUploader(api)
	existingObjectId, err := uploader.getObjectIdIfAlreadyExists(d, api, api.GetUrlFromEnvironmentUrl(d.environmentUrl), name)
	return existingObjectId != "", existingObjectId, err
}

func (d *dynatraceClientImpl) UpsertByName(api Api, name string, payload []byte) (entity DynatraceEntity, err error) {

	fullUrl := api.GetUrlFromEnvironmentUrl(d.environmentUrl)

	if api.GetId() == "extension" {
		return uploadExtension(d.client, fullUrl, name, payload, d.token)
	}

	uploader := newConfigUploader(api)
	return uploader.upsertDynatraceObject(d, fullUrl, name, api, payload, false)
}

func (d *dynatraceClientImpl) ValidateByName(api Api, name string, payload []byte) (valid bool, err error) {

	if !api.IsSettings20Api() {
		return true, nil
	}

	fullUrl := api.GetUrlFromEnvironmentUrl(d.environmentUrl)
	uploader := newConfigUploader(api)

	_, err = uploader.upsertDynatraceObject(d, fullUrl, name, api, payload, true)
	return err != nil, err
}
