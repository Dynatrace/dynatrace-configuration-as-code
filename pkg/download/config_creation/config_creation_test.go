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

type uidStruct struct {
	Uid string `json:"uid"`
}

type idStruct struct {
	Id string `json:"id"`
}

func TestCreateConfig(t *testing.T) {
	t.Run("it returns an error if JSON is invalid", func(t *testing.T) {
		myStruct := uidStruct{}
		preparedConfig, err := config_creation.PrepareConfig([]byte(`{"uid": "my-id","name": "my-resource"`) /*missing "}" at the end*/, &myStruct, []string{"version", "externalId"}, "")

		assert.Equal(t, "", myStruct.Uid)
		assert.Equal(t, "", preparedConfig.JSONString)
		assert.EqualError(t, err, "failed to unmarshal payload: unexpected end of JSON input")
	})

	t.Run("it returns a valid SLO configuration without externalId, id and version", func(t *testing.T) {
		myStruct := idStruct{}
		preparedConfig, err := config_creation.PrepareConfig([]byte(`{"id": "my-id", "externalId": "e-id", "version": "xy", "name": "my-resource"}`), &myStruct, []string{"id", "version", "externalId"}, "")

		assert.NoError(t, err)
		assert.Equal(t, "my-id", myStruct.Id)
		assert.Equal(t, "{\n  \"name\": \"my-resource\"\n}", preparedConfig.JSONString)
	})
	t.Run("it returns a valid segment configuration without uid, version and externalId", func(t *testing.T) {
		myStruct := uidStruct{}

		preparedConfig, err := config_creation.PrepareConfig([]byte(`{"uid": "my-uid", "version": "xy", "externalId": "e-id", "name": "my-resource"}`), &myStruct, []string{"uid", "version", "externalId"}, "")

		assert.NoError(t, err)
		assert.Equal(t, "my-uid", myStruct.Uid)
		assert.Equal(t, "{\n  \"name\": \"my-resource\"\n}", preparedConfig.JSONString)
	})
	t.Run("it returns and replaces parameters if given", func(t *testing.T) {
		myStruct := uidStruct{}

		preparedConfig, err := config_creation.PrepareConfig([]byte(`{"uid": "my-uid", "version": "xy", "externalId": "e-id", "name": "my-resource"}`), &myStruct, []string{"uid", "version", "externalId"}, "name")

		assert.NoError(t, err)
		assert.Equal(t, "my-uid", myStruct.Uid)
		assert.Equal(t, "{\n  \"name\": \"{{.name}}\"\n}", preparedConfig.JSONString)
	})
}
