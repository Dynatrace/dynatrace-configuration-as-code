// @license
// Copyright 2021 Dynatrace LLC
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

//go:build unit
// +build unit

package jsoncreator

import (
	"encoding/json"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

func TestCreateJsonConfig(t *testing.T) {
	jsonsample := []byte("{ \"name\": \"test1\"}")

	apiMock := api.CreateAPIMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	fs := util.CreateTestFileSystem()
	val := api.Value{Id: "acc3c230-e156-4a11-a5b7-bda1b304e613", Name: "Sockshop Error Profile"}
	client.
		EXPECT().
		ReadById(apiMock, val.Id).
		Return(jsonsample, nil)

	apiMock.EXPECT().GetId().Return("alerting-profile").AnyTimes()
	apiMock.EXPECT().IsNonUniqueNameApi().Return(false).AnyTimes()

	jcreator := NewJSONCreator()

	filter, err := jcreator.CreateJSONConfig(fs, client, apiMock, val.Id, "/")
	assert.NilError(t, err)
	assert.Equal(t, filter, false)
}

func TestIsDefaultEntityDashboardCase(t *testing.T) {
	//create payload similar to dynatrace API object for dashboard
	sample := make(map[string]interface{})
	sample["dashboardMetadata"] = make(map[string]interface{})
	metadata := sample["dashboardMetadata"].(map[string]interface{})
	metadata["preset"] = true

	result := isDefaultEntity("dashboard", sample)
	assert.Equal(t, result, false)

	metadata["owner"] = "Dynatrace"
	result = isDefaultEntity("dashboard", sample)
	assert.Equal(t, result, true)

	result = isDefaultEntity("synthetic-location", sample)
	assert.Equal(t, result, true)

	result = isDefaultEntity("synthetic-monitor", sample)
	assert.Equal(t, result, false)

	result = isDefaultEntity("extension", sample)
	assert.Equal(t, result, false)

	result = isDefaultEntity("aws-credentials", sample)
	assert.Equal(t, result, false)
}

func TestIsDefaultEntityHostsAutoUpdateCase(t *testing.T) {
	// Create payload similar to dynatrace API object
	sample := make(map[string]interface{})

	jsonBlob := []byte(`{
		"setting": "DISABLED",
		"targetVersion": null,
		"updateWindows": {
			"windows": []
		},
		"version": null
	 }`)
	json.Unmarshal(jsonBlob, &sample)

	result := isDefaultEntity("hosts-auto-update", sample)
	assert.Equal(t, result, true)

	jsonBlob = []byte(`{
		"setting": "ENABLED",
		"targetVersion": null,
		"updateWindows": {
			"windows": [{
				"id": "existing-update-window"
			}]
		},
		"version": null
	 }`)
	json.Unmarshal(jsonBlob, &sample)

	result = isDefaultEntity("hosts-auto-update", sample)
	assert.Equal(t, result, false)
}

func TestProcessJSONFile(t *testing.T) {
	sample := make(map[string]interface{})
	sample["testprop"] = "testprop"
	sample["name"] = "test1"
	sample["displayName"] = "testDisplay"
	sample["id"] = "testId"

	apiMock := api.CreateAPIMockFactory(t)
	apiMock.EXPECT().GetId().Return("alerting-profile").AnyTimes()
	apiMock.EXPECT().IsNonUniqueNameApi().Return(false).Times(1)

	file, err := processJSONFile(sample, "testId")
	assert.NilError(t, err)

	jsonfile := make(map[string]interface{})
	err = json.Unmarshal(file, &jsonfile)
	assert.Check(t, jsonfile["testprop"] == "testprop")
	assert.Check(t, jsonfile["name"] == "{{.name}}")
	assert.Check(t, jsonfile["displayName"] == "{{.name}}")
	assert.Check(t, jsonfile["id"] == nil)

	apiMock.EXPECT().IsNonUniqueNameApi().Return(true).Times(1)

	_, err = processJSONFile(sample, "testId")
	assert.NilError(t, err)
}

func TestReplaceKeyProperties(t *testing.T) {
	data := map[string]interface{}{
		"metadata":    true,
		"id":          12,
		"identifier":  "id",
		"entityId":    "aaa",
		"noRemove":    "aaa",
		"name":        "replace",
		"displayName": "replace",
		"rules": []interface{}{
			map[string]interface{}{
				"2":           12,
				"id":          "random string",
				"methodRules": map[string]interface{}{"bool": false, "id": "asdf"},
				"bool":        false,
			},
			map[string]interface{}{
				"2":           12,
				"id":          "random string",
				"methodRules": map[string]interface{}{"id": "asdf"},
				"bool":        false,
			},
		},
	}
	data = replaceKeyProperties(data)

	assert.Assert(t, data["metadata"] == nil, "metadata should be removed")
	assert.Assert(t, data["id"] == nil, "id should be removed")
	assert.Assert(t, data["entityId"] == nil, "entityId should be removed")
	assert.Assert(t, data["identifier"] == nil, "identifier should be removed")
	assert.Assert(t, data["rules"] != nil, "rules should exist and not be removed")
	assert.Assert(t, len(data) == 4, "too many or too little elements have been removed")
	assert.Assert(t, data["rules"].([]interface{})[0].(map[string]interface{})["id"] == nil, "rule.id should be removed")
	assert.Assert(t, data["rules"].([]interface{})[1].(map[string]interface{})["id"] == nil, "rule.id should be removed even if there is not just one")
	assert.Assert(t, data["rules"].([]interface{})[0].(map[string]interface{})["bool"] != nil, "rule.bool must not be removed")
	assert.Assert(t, data["rules"].([]interface{})[0].(map[string]interface{})["methodRules"] != nil, "rule.methodRules must exist")
	assert.Assert(t, data["rules"].([]interface{})[1].(map[string]interface{})["methodRules"] != nil, "rule.methodRules must exist")
	assert.Assert(t, data["rules"].([]interface{})[0].(map[string]interface{})["methodRules"].(map[string]interface{})["id"] == nil, "ruel.methodRules.id should be removed")
	assert.Assert(t, data["rules"].([]interface{})[1].(map[string]interface{})["methodRules"].(map[string]interface{})["id"] == nil, "ruel.methodRules.id should be removed")
	assert.Assert(t, data["rules"].([]interface{})[0].(map[string]interface{})["methodRules"].(map[string]interface{})["bool"] != nil, "ruel.methodRules.bool must not be removed")

	//replace test
	assert.Assert(t, data["name"] == "{{.name}}", "names have to be replaced")
	assert.Assert(t, data["displayName"] == "{{.name}}", "names have to be replaced")
	assert.Assert(t, data["dashboardId"] == nil, "nonexistent values should not be created")
}

func TestRemoveKey_TooLittleRemoved(t *testing.T) {
	data := map[string]interface{}{
		"a":  true,
		"id": 12,
		"b":  "AAA",
		"c":  "aaa",
		"d":  "A",
		"e":  "A",
		"REC": []interface{}{
			map[string]interface{}{
				"a":    12,
				"b":    "random string",
				"rec2": map[string]interface{}{"bool": false, "id": "asdf"},
				"bool": false,
			},
			map[string]interface{}{
				"a": 12,
				"b": "random string",
				"c": map[string]interface{}{"id": "asdf"},
				"d": false,
			},
		},
	}

	data = removeKey(data, []string{"id"})
	assert.Assert(t, data["id"] == nil, "id must be removed")

	data = removeKey(data, []string{"REC", "a"})
	assert.Assert(t, data["REC"].([]interface{})[0].(map[string]interface{})["a"] == nil, "must be removed")
	assert.Assert(t, data["REC"].([]interface{})[1].(map[string]interface{})["a"] == nil, "must be removed")

	data = removeKey(data, []string{"REC", "rec2"})
	assert.Assert(t, data["REC"].([]interface{})[0].(map[string]interface{})["rec2"] == nil, "must be removed")
	assert.Assert(t, data["REC"].([]interface{})[1].(map[string]interface{})["rec2"] == nil, "must be removed")

}

func TestRemoveKey_TooMuchRemoved(t *testing.T) {
	data := map[string]interface{}{
		"a":  true,
		"id": 12,
		"b":  "AAA",
		"c":  "aaa",
		"d":  "A",
		"e":  "A",
		"REC": []interface{}{
			map[string]interface{}{
				"a":    12,
				"b":    "random string",
				"rec2": map[string]interface{}{"bool": false, "id": "asdf"},
				"bool": false,
			},
			map[string]interface{}{
				"a": 12,
				"b": "random string",
				"c": map[string]interface{}{"id": "asdf"},
				"d": false,
			},
		},
	}

	data = removeKey(data, []string{"invalid"})
	assert.Equal(t, len(data), 7, "data must not change")

	data = removeKey(data, []string{"not exist", "a"})
	assert.Equal(t, len(data), 7, "data must not change")

	data = removeKey(data, []string{"a", "a"})
	assert.Equal(t, len(data), 7, "data must not change")

}
