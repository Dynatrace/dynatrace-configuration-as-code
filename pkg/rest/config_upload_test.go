//go:build unit
// +build unit

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
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestTranslateGenericValuesOnStandardResponse(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(response, "extensions")

	assert.NilError(t, err)
	assert.Check(t, len(values) == 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "bar")
}

func TestTranslateGenericValuesOnIdMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["name"] = "bar"

	response := make([]interface{}, 1)
	response[0] = entry

	_, err := translateGenericValues(response, "extensions")

	assert.ErrorContains(t, err, "config of type extensions was invalid: No id")
}

func TestTranslateGenericValuesOnNameMissing(t *testing.T) {

	entry := make(map[string]interface{})
	entry["id"] = "foo"

	response := make([]interface{}, 1)
	response[0] = entry

	values, err := translateGenericValues(response, "extensions")

	assert.NilError(t, err)
	assert.Check(t, len(values) == 1)

	assert.Equal(t, values[0].Id, "foo")
	assert.Equal(t, values[0].Name, "foo")
}

func Test_replaceNameWithGeneratedUuid(t *testing.T) {
	type args struct {
		objectName string
		payload    []byte
	}
	tests := []struct {
		name                string
		givenName           string
		givenPayload        []byte
		wantUuid            string
		wantModifiedPayload []byte
		wantErr             bool
	}{
		{
			"replacesNamePropertyWithUUid",
			"an application detection rule",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\": \"an application detection rule\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\": \"51f47928-d86a-3cd0-9a2a-b0f04a1c4531\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			false,
		},
		{
			"replacesNamePropertyWithUUid_withoutWhitespaces",
			"an application detection rule",
			[]byte("{\n  \"applicationIdentifier\":\"42\",\n\"name\":\"an application detection rule\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
			[]byte("{\n  \"applicationIdentifier\":\"42\",\n\"name\": \"51f47928-d86a-3cd0-9a2a-b0f04a1c4531\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			false,
		},
		{
			"replacesNamePropertyWithUUid_withWhitespaces",
			"an application detection rule",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\" :  \"an application detection rule\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\": \"51f47928-d86a-3cd0-9a2a-b0f04a1c4531\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			false,
		},
		{
			"replacesNamePropertyWithUUid_withMoreWhitespaces",
			"an application detection rule",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\"    :          \"an application detection rule\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
			[]byte("{\n  \"applicationIdentifier\": \"42\",\n  \"name\": \"51f47928-d86a-3cd0-9a2a-b0f04a1c4531\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			false,
		},
		{
			"replacesNamePropertyWithUUid_butNoOtherOccurrences",
			"an application detection rule",
			[]byte("{\n  \"applicationIdentifier\": \"an application detection rule\",\n  \"name\": \"an application detection rule\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
			[]byte("{\n  \"applicationIdentifier\": \"an application detection rule\",\n  \"name\": \"51f47928-d86a-3cd0-9a2a-b0f04a1c4531\",\n  \"filterConfig\": {\n    \"pattern\": \"A pattern\",\n    \"applicationMatchType\": \"BEGINS_WITH\",\n    \"applicationMatchTarget\": \"URL\"\n  }\n}"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUuid, gotPayload, err := replaceNameWithGeneratedUuid(tt.givenName, tt.givenPayload)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceNameWithGeneratedUuid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUuid != tt.wantUuid {
				t.Errorf("replaceNameWithGeneratedUuid() gotUuid = %v, want %v", gotUuid, tt.wantUuid)
			}
			if !reflect.DeepEqual(gotPayload, tt.wantModifiedPayload) {
				t.Errorf("replaceNameWithGeneratedUuid() gotPayload = %v, want %v", string(gotPayload), string(tt.wantModifiedPayload))
			}
		})
	}
}
