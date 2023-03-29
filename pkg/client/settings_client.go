// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/rest"
)

// SettingsObject contains all the information necessary to create/update a settings object
type SettingsObject struct {
	// Id is the monaco related Configuration ID
	Id,
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

type settingsRequest struct {
	SchemaId      string `json:"schemaId"`
	ExternalId    string `json:"externalId,omitempty"`
	Scope         string `json:"scope"`
	Value         any    `json:"value"`
	SchemaVersion string `json:"schemaVersion,omitempty"`
	ObjectId      string `json:"objectId,omitempty"`
}

// buildPostRequestPayload builds the json that is required as body in the settings api.
// POST Request body: https://www.dynatrace.com/support/help/dynatrace-api/environment-api/settings/objects/post-object#request-body-json-model
//
// To do this, we have to wrap the template in another object and send this object to the server.
// Currently, we only encode one object into an array of objects, but we can optimize it to contain multiple elements to update.
// Note payload limitations: https://www.dynatrace.com/support/help/dynatrace-api/basics/access-limit#payload-limit
func buildPostRequestPayload(obj SettingsObject, externalId string) ([]byte, error) {
	var value any
	if err := json.Unmarshal(obj.Content, &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rendered config: %w", err)
	}

	data := settingsRequest{
		SchemaId:      obj.SchemaId,
		ExternalId:    externalId,
		Scope:         obj.Scope,
		Value:         value,
		SchemaVersion: obj.SchemaVersion,
		ObjectId:      obj.OriginObjectId,
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
		log.Debug("Failed to compact json: %w. Using uncompressed json.\n\tJson: %v", err, string(fullObj))
		return fullObj, nil
	}

	return dest.Bytes(), nil
}

type postResponse struct {
	ObjectId string `json:"objectId"`
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
