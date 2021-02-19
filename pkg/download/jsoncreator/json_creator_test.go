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

// +build unit

package jsoncreator

import (
	"encoding/json"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func TestCreateJsonConfig(t *testing.T) {
	jsonsample := []byte("{ \"name\": \"test1\"}")

	apiMock := api.CreateAPIMockFactory(t)
	creator := files.CreateFileCreatorMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	val := api.Value{Id: "acc3c230-e156-4a11-a5b7-bda1b304e613", Name: "Sockshop Error Profile"}
	client.
		EXPECT().
		ReadById(apiMock, val.Id).
		Return(jsonsample, nil)

	apiMock.EXPECT().GetId().Return("alerting-profile").AnyTimes()

	creator.
		EXPECT().
		CreateFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("alerting-profile.json", nil)
	jcreator := NewJSONCreator()

	name, filter, err := jcreator.CreateJSONConfig(client, apiMock, val, creator, "/")
	assert.NilError(t, err)
	assert.Equal(t, filter, false)
	assert.Equal(t, name, "alerting-profile.json")
}
func TestIsDefaultEntityDashboardCase(t *testing.T) {
	//create payload similar to dynatrace API object for dashboard
	sample := make(map[string]interface{})
	sample["dashboardMetadata"] = make(map[string]interface{})
	metadata := sample["dashboardMetadata"].(map[string]interface{})
	metadata["preset"] = true
	result := isDefaultEntity("dashboard", sample)
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

func TestProcessJSONFile(t *testing.T) {
	sample := make(map[string]interface{})
	sample["testprop"] = "testprop"
	sample["name"] = "test1"
	sample["displayName"] = "testDisplay"
	sample["id"] = "testId"
	file, err := processJSONFile(sample, "testId")
	assert.NilError(t, err)
	jsonfile := make(map[string]interface{})
	err = json.Unmarshal(file, &jsonfile)
	assert.Check(t, jsonfile["testprop"] == "testprop")
	assert.Check(t, jsonfile["name"] == "{{.name}}")
	assert.Check(t, jsonfile["displayName"] == "{{.name}}")
	assert.Check(t, jsonfile["id"] == nil)
}
