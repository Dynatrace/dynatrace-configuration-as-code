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

	name, cleanName, filter, err := jcreator.CreateJSONConfig(fs, client, apiMock, val, "/")
	assert.NilError(t, err)
	assert.Equal(t, filter, false)
	assert.Equal(t, name, "Sockshop Error Profile")
	assert.Equal(t, cleanName, "SockshopErrorProfile")
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

	file, name, cleanName, err := processJSONFile(sample, "testId", "test1", apiMock)
	assert.NilError(t, err)

	jsonfile := make(map[string]interface{})
	err = json.Unmarshal(file, &jsonfile)
	assert.Check(t, jsonfile["testprop"] == "testprop")
	assert.Check(t, jsonfile["name"] == "{{.name}}")
	assert.Check(t, jsonfile["displayName"] == "{{.name}}")
	assert.Check(t, name == "test1")
	assert.Check(t, cleanName == "test1")
	assert.Check(t, jsonfile["id"] == nil)

	apiMock.EXPECT().IsNonUniqueNameApi().Return(true).Times(1)

	_, _, cleanName, err = processJSONFile(sample, "testId", "test1", apiMock)
	assert.NilError(t, err)
	assert.Equal(t, "testId", cleanName)
}
