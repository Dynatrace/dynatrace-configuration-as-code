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

package config_creation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/config_creation"
)

func TestCreateConfig_Error(t *testing.T) {
	testcases := []struct {
		Name  string
		IdKey string
		Data  []byte
		Error string
	}{
		{
			Name:  "Returns an error if JSON is invalid",
			IdKey: "id",
			Data:  []byte(`{"id": "my-id","name": "my-resource"`), // missing "}" at the end
			Error: "failed to unmarshal payload: unexpected end of JSON input",
		},
		{
			Name:  "Returns an error if the given ID is missing",
			IdKey: "uid",
			Data:  []byte(`{"id": "my-id", "name": "my-resource"}`),
			Error: "API payload is missing 'uid'",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			id, jsonString, err := config_creation.PrepareConfig(testcase.Data, testcase.IdKey, testcase.IdKey, "version", "externalId")

			assert.Equal(t, "", id)
			assert.Equal(t, "", jsonString)
			assert.EqualError(t, err, testcase.Error)
		})
	}
}

func TestCreateConfig_Result(t *testing.T) {
	testcases := []struct {
		Name       string
		IdKey      string
		Data       []byte
		ResultId   string
		ResultJSON string
	}{
		{
			Name:       "Returns a valid SLO configuration without externalId, id and version",
			IdKey:      "id",
			Data:       []byte(`{"id": "my-id", "externalId": "e-id", "version": "xy", "name": "my-resource"}`),
			ResultId:   "my-id",
			ResultJSON: "{\n  \"name\": \"my-resource\"\n}",
		},
		{
			Name:       "Returns a valid segment configuration without uid, version and externalId",
			IdKey:      "uid",
			Data:       []byte(`{"uid": "my-uid", "version": "xy", "externalId": "e-id", "name": "my-resource"}`),
			ResultId:   "my-uid",
			ResultJSON: "{\n  \"name\": \"my-resource\"\n}",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			id, resultJson, err := config_creation.PrepareConfig(testcase.Data, testcase.IdKey, testcase.IdKey, "version", "externalId")

			assert.NoError(t, err)
			assert.Equal(t, testcase.ResultId, id)
			assert.Equal(t, testcase.ResultJSON, resultJson)
		})
	}
}
