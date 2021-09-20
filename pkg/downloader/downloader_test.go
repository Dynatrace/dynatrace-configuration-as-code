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

package downloader

// func TestCreateConfigsFromAPI(t *testing.T) {
// 	apiMock := api.CreateAPIMockFactory(t)
// 	client := rest.CreateDynatraceClientMockFactory(t)
// 	jcreator := jsoncreator.CreateJSONCreatorMock(t)
// 	ycreator := yamlcreator.CreateYamlCreatorMock(t)
// 	fs := util.CreateTestFileSystem()
// 	list := []api.Value{{Id: "d", Name: "namevalue"}}

// 	client.EXPECT().
// 		List(gomock.Any()).Return(list, nil)

// 	apiMock.EXPECT().
// 		GetId().Return("synthetic-monitor").AnyTimes()

// 	jcreator.EXPECT().
// 		CreateJSONConfig(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
// 		Return("demo.json", "demo", false, nil)

// 	ycreator.EXPECT().
// 		CreateYamlFile(gomock.Any(), gomock.Any(), gomock.Any()).
// 		Return(nil)
// 	ycreator.EXPECT().AddConfig(gomock.Any(), gomock.Any())

// 	err := createConfigsFromAPI(fs, apiMock, "123", "/", client, jcreator, ycreator)
// 	assert.NilError(t, err, "No errors")
// }

// func TestDownloadConfigFromEnvironment(t *testing.T) {
// 	// os.Setenv("token", "test")
// 	// env := environment.NewEnvironment("environment1", "test", "", "https://test.live.dynatrace.com", "token")

// 	// fileManager := util.CreateTestFileSystem()
// 	// err := DownloadConfigFromEnvironment(fileManager, env, "", nil)
// 	// assert.NilError(t, err)
// }
