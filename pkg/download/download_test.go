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

package download

import (
	"os"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/jsoncreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/download/yamlcreator"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

func TestGetConfigs(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")
	envs := make(map[string]environment.Environment)
	fileManager := util.CreateTestFileSystem()
	envs["e1"] = env
	err := getConfigs(fileManager, "", envs, "")
	assert.ErrorContains(t, err, "There were some errors")
}

func TestCreateConfigsFromAPI(t *testing.T) {
	apiMock := api.CreateAPIMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	jcreator := jsoncreator.CreateJSONCreatorMock(t)
	ycreator := yamlcreator.CreateYamlCreatorMock(t)
	fs := util.CreateTestFileSystem()
	list := []api.Value{{Id: "d", Name: "namevalue"}}

	client.EXPECT().
		List(gomock.Any()).Return(list, nil)

	apiMock.EXPECT().
		GetId().Return("synthetic-monitor").AnyTimes()

	apiMock.EXPECT().
		IsNonUniqueNameApi().Return(false).AnyTimes()

	apiMock.EXPECT().
		IsSingleConfigurationApi().Return(false).AnyTimes()

	jcreator.EXPECT().
		CreateJSONConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(false, nil)

	ycreator.
		EXPECT().
		ReadYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.
		EXPECT().
		UpdateConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	ycreator.
		EXPECT().
		WriteYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.
		EXPECT().
		GetConfigFileName(gomock.Any()).
		Return("")

	err := createConfigsFromAPI(fs, apiMock, "/", client, jcreator, ycreator)
	assert.NilError(t, err, "No errors")

	dirCreated, err := afero.DirExists(fs, "/synthetic-monitor")
	assert.NilError(t, err, "No errors")
	assert.Equal(t, dirCreated, true, "no folder got created")

}

func TestDownloadConfigFromEnvironment(t *testing.T) {
	os.Setenv("token", "test")
	env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")

	fileManager := util.CreateTestFileSystem()
	err := downloadConfigFromEnvironment(fileManager, env, "", nil)
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

func TestCreateConfigsFromAPI_NoJSONCreated_NoEmptyFolder(t *testing.T) {
	apiMock := api.CreateAPIMockFactory(t)
	client := rest.CreateDynatraceClientMockFactory(t)
	jcreator := jsoncreator.CreateJSONCreatorMock(t)
	ycreator := yamlcreator.CreateYamlCreatorMock(t)
	fs := util.CreateTestFileSystem()
	list := []api.Value{{Id: "d", Name: "namevalue"}}

	client.EXPECT().
		List(gomock.Any()).Return(list, nil)

	apiMock.EXPECT().
		GetId().Return("synthetic-monitor").AnyTimes()

	apiMock.EXPECT().
		IsNonUniqueNameApi().Return(false).AnyTimes()

	apiMock.EXPECT().
		IsSingleConfigurationApi().Return(false).AnyTimes()

	jcreator.EXPECT().
		CreateJSONConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(true, nil)

	ycreator.
		EXPECT().
		ReadYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.
		EXPECT().
		UpdateConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())
	ycreator.
		EXPECT().
		WriteYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)
	ycreator.
		EXPECT().
		GetConfigFileName(gomock.Any()).
		Return("")

	err := createConfigsFromAPI(fs, apiMock, "/", client, jcreator, ycreator)
	assert.NilError(t, err, "error in createConfigsFromAPI")

	dirCreated, err := afero.DirExists(fs, "/synthetic-monitor") // no folder is created
	assert.NilError(t, err, "error in fake File system")
	assert.Equal(t, dirCreated, false, "folder got created")
}
