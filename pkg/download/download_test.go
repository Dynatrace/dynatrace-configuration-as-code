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

package download

import (
	"os"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/jsoncreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/yamlcreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/files"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
)

func TestGetConfigs(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")
	envs := make(map[string]environment.Environment)
	envs["e1"] = env
	err := getConfigs("", envs, "")
	assert.NilError(t, err)
}
func TestCreateConfigsFromAPI(t *testing.T) {
	apiMock := api.CreateAPIMockFactory(t)
	fcreator := files.CreateFileCreatorMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	jcreator := jsoncreator.CreateJSONCreatorMock(t)
	ycreator := yamlcreator.CreateYamlCreatorMock(t)

	list := []api.Value{{Id: "d", Name: "namevalue"}}

	client.EXPECT().
		List(gomock.Any()).Return(list, nil)

	apiMock.EXPECT().
		GetId().Return("synthetic-monitor").AnyTimes()

	jcreator.EXPECT().
		CreateJSONConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("demo.json", false, nil)

	ycreator.EXPECT().
		CreateYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.EXPECT().AddConfig(gomock.Any(), gomock.Any())
	fcreator.EXPECT().
		CreateFolder("/synthetic-monitor").
		Return("synthetic-monitor", nil)

	err := createConfigsFromAPI(apiMock, "123", fcreator, "/", client, jcreator, ycreator)
	assert.NilError(t, err, "No errors")
}

func TestDownloadConfigFromEnvironment(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")
	err := downloadConfigFromEnvironment(env, "", nil)
	assert.NilError(t, err)
}
func TestGetAPIList(t *testing.T) {
	//multiple options
	list, err := getAPIList("synthetic-location,   extension, alerting-profile")
	assert.NilError(t, err)
	assert.Check(t, list["synthetic-location"].GetId() == "synthetic-location")
	assert.Check(t, list["dashboard"] == nil)
	list, err = getAPIList("synthetic-location,extension,dashboard")
	assert.NilError(t, err)
	//single option
	list, err = getAPIList("synthetic-location")
	assert.NilError(t, err)
	//no option
	list, err = getAPIList("")
	assert.NilError(t, err)
	list, err = getAPIList(" ")
	assert.NilError(t, err)
	//not a real API
	list, err = getAPIList("synthetic-location-test,   extension-test, alerting-profile")
	assert.ErrorContains(t, err, "There were some errors in the API list provided")
}
