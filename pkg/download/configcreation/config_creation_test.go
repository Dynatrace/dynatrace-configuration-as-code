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

package configcreation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/download/configcreation"
)

type uidStruct struct {
	Uid string `json:"uid"`
}

type idStruct struct {
	Id string `json:"id"`
}

func TestCreateConfig_Error(t *testing.T) {
	basicString := "asdf"
	myStruct := uidStruct{}

	testcases := []struct {
		Name       string
		Data       []byte
		WantStruct any
		WantError  string
	}{
		{
			Name:       "it returns an error if JSON is invalid",
			Data:       []byte(`{"uid": "my-id","name": "my-resource"`), /*missing "}" at the end*/
			WantStruct: &myStruct,
			WantError:  "failed to unmarshal payload: unexpected end of JSON input",
		},
		{
			Name: "it returns an error if the struct is not a pointer", Data: []byte(`{"uid": "my-id","name": "my-resource"}`),
			WantStruct: myStruct,
			WantError:  "failed to unmarshal payload: json: Unmarshal(non-pointer configcreation_test.uidStruct)",
		},
		{
			Name: "it returns an error if the struct is just a string", Data: []byte(`{"uid": "my-id","name": "my-resource"}`),
			WantStruct: "asdf",
			WantError:  "failed to unmarshal payload: json: Unmarshal(non-pointer string)",
		},
		{
			Name:       "it returns an error if the struct is just a string pointer",
			Data:       []byte(`{"uid": "my-id","name": "my-resource"}`),
			WantStruct: &basicString,
			WantError:  "failed to unmarshal payload: json: cannot unmarshal object into Go value of type string",
		},
	}
	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			preparedConfig, err := configcreation.PrepareConfig(testcase.Data, testcase.WantStruct, []string{}, "")

			assert.Equal(t, "", preparedConfig.JSONString)
			assert.EqualError(t, err, testcase.WantError)
		})
	}
}

func TestCreateConfig_Valid(t *testing.T) {
	t.Run("it returns a valid SLO configuration without externalId, id and version", func(t *testing.T) {
		myStruct := idStruct{}
		preparedConfig, err := configcreation.PrepareConfig([]byte(`{"id": "my-id", "externalId": "e-id", "version": "xy", "name": "my-resource"}`), &myStruct, []string{"id", "version", "externalId"}, "")

		assert.NoError(t, err)
		assert.Equal(t, "my-id", myStruct.Id)
		assert.Equal(t, "{\n  \"name\": \"my-resource\"\n}", preparedConfig.JSONString)
	})

	t.Run("it returns a valid segment configuration without uid, version and externalId", func(t *testing.T) {
		myStruct := uidStruct{}

		preparedConfig, err := configcreation.PrepareConfig([]byte(`{"uid": "my-uid", "version": "xy", "externalId": "e-id", "name": "my-resource"}`), &myStruct, []string{"uid", "version", "externalId"}, "")

		assert.NoError(t, err)
		assert.Equal(t, "my-uid", myStruct.Uid)
		assert.Equal(t, "{\n  \"name\": \"my-resource\"\n}", preparedConfig.JSONString)
	})

	t.Run("it returns and replaces parameters if given", func(t *testing.T) {
		myStruct := uidStruct{}

		preparedConfig, err := configcreation.PrepareConfig([]byte(`{"uid": "my-uid", "version": "xy", "externalId": "e-id", "name": "my-resource"}`), &myStruct, []string{"uid", "version", "externalId"}, "name")

		assert.NoError(t, err)
		assert.Equal(t, "my-uid", myStruct.Uid)
		assert.Equal(t, "{\n  \"name\": \"{{.name}}\"\n}", preparedConfig.JSONString)
		assert.Equal(t, map[string]parameter.Parameter{"name": &value.ValueParameter{Value: "my-resource"}}, preparedConfig.Parameters)
	})
}
