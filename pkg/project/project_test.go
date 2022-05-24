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

package project

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

func testCreateProjectBuilder(projectsRoot string) projectBuilder {

	return projectBuilder{
		projectRootFolder: projectsRoot,
		apis:              createTestApis(),
		configFactory:     config.NewConfigFactory(),
		configs:           make([]config.Config, 10),
	}
}

func testCreateProjectBuilderWithMock(factory config.ConfigFactory, fs afero.Fs, projectId string, projectsRoot string) projectBuilder {

	return projectBuilder{
		projectRootFolder: projectsRoot,
		projectId:         projectId,
		apis:              createTestApis(),
		configs:           make([]config.Config, 0),
		configFactory:     factory,
		fs:                fs,
	}
}

func createTestApis() map[string]api.Api {

	apis := make(map[string]api.Api)
	apis["alerting-profile"] = testAlertingProfileApi
	apis["management-zone"] = testManagementZoneApi
	apis["dashboard"] = testDashboardApi

	return apis
}

var testAlertingProfileApi = api.NewStandardApi("alerting-profile", "/api/config/v1/alertingProfiles", false)
var testManagementZoneApi = api.NewStandardApi("management-zone", "/api/config/v1/managementZones", false)
var testDashboardApi = api.NewStandardApi("dashboard", "/api/config/v1/dashboards", true)

func TestGetPathSuccess(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := util.ReplacePathSeparators("management-zone/testytest.json")

	err, configType := builder.getConfigTypeFromLocation(json)

	assert.NilError(t, err)
	assert.Equal(t, "management-zone", configType.GetId())

	err, configType = builder.getConfigTypeFromLocation(json)

	assert.NilError(t, err)
	assert.Equal(t, "management-zone", configType.GetId())
}

func TestGetPathTooLittleArgs(t *testing.T) {

	builder := testCreateProjectBuilder("")
	err, _ := builder.getConfigTypeFromLocation("testytest.json")

	assert.Error(t, err, "path testytest.json too short")
}

func TestRemoveYamlFromPath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	yaml := util.ReplacePathSeparators("project/dashboards/config.yaml")
	expected := util.ReplacePathSeparators("project/dashboards")
	err, result := builder.removeYamlFileFromPath(yaml)

	assert.NilError(t, err)
	assert.Equal(t, expected, result)
}

func TestRemoveYamlFromPathWhenPathIsTooShort(t *testing.T) {

	builder := testCreateProjectBuilder("")
	err, _ := builder.removeYamlFileFromPath("config.yaml")

	assert.Error(t, err, "path config.yaml too short")
}

func TestGetApiInformationFromLocation(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := util.ReplacePathSeparators("test/management-zone/testytest.json")
	err, apiInfo := builder.getExtendedInformationFromLocation(json)

	assert.NilError(t, err)
	assert.Equal(t, testManagementZoneApi, apiInfo)
}

func TestGetConfigTypeInformationFromLocation(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := util.ReplacePathSeparators("test/alerting-profile/testytest.json")
	json1 := util.ReplacePathSeparators("cluster/test/alerting-profile/testytest.json")
	json2 := util.ReplacePathSeparators("config/cluster/test/alerting-profile/testytest.json")
	err, api := builder.getExtendedInformationFromLocation(json)
	err1, api1 := builder.getExtendedInformationFromLocation(json1)
	err2, api2 := builder.getExtendedInformationFromLocation(json2)

	assert.NilError(t, err)
	assert.NilError(t, err1)
	assert.NilError(t, err2)
	assert.Equal(t, "alerting-profile", api.GetId())
	assert.Equal(t, "alerting-profile", api1.GetId())
	assert.Equal(t, "alerting-profile", api2.GetId())
}

func TestGetApiFromLocationApiNotFound(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := util.ReplacePathSeparators("test/notexisting/testytest.json")
	err, _ := builder.getExtendedInformationFromLocation(json)

	assert.ErrorContains(t, err, "API was unknown")
}

func TestProcessConfigSection(t *testing.T) {

	factory := config.CreateConfigMockFactory(t)
	fs := util.CreateTestFileSystem()
	builder := testCreateProjectBuilderWithMock(factory, fs, "testProject", "")

	m := make(map[string]map[string]string)

	m["config"] = make(map[string]string)

	m["config"]["test1"] = util.ReplacePathSeparators("/test/management-zone/zoneA.json")
	m["config"]["test2"] = util.ReplacePathSeparators("/test/alerting-profile/profile.json")

	zoneA := util.ReplacePathSeparators("test/management-zone/zoneA.json")
	profile := util.ReplacePathSeparators("test/alerting-profile/profile.json")
	factory.EXPECT().NewConfig(fs, "test1", "testProject", zoneA, m, testManagementZoneApi).Times(1)
	factory.EXPECT().NewConfig(fs, "test2", "testProject", profile, m, testAlertingProfileApi).Times(1)

	folderPath := util.ReplacePathSeparators("test/management-zone")
	err := builder.processConfigSection(m, folderPath)
	assert.NilError(t, err)
}

func TestProcessConfigSectionWithProjectRootParameter(t *testing.T) {

	factory := config.CreateConfigMockFactory(t)
	fileReaderMock := util.CreateTestFileSystem()
	builder := testCreateProjectBuilderWithMock(factory, fileReaderMock, "test", "testProjectsRoot")

	m := make(map[string]map[string]string)

	m["config"] = make(map[string]string)

	m["config"]["testconfig1"] = util.ReplacePathSeparators("/test/management-zone/zoneA.json")
	m["config"]["testconfig2"] = util.ReplacePathSeparators("/test/alerting-profile/profile.json")

	zoneA := util.ReplacePathSeparators("testProjectsRoot/test/management-zone/zoneA.json")
	profile := util.ReplacePathSeparators("testProjectsRoot/test/alerting-profile/profile.json")
	factory.EXPECT().NewConfig(fileReaderMock, "testconfig1", "test", zoneA, m, testManagementZoneApi).Times(1)
	factory.EXPECT().NewConfig(fileReaderMock, "testconfig2", "test", profile, m, testAlertingProfileApi).Times(1)

	folderPath := util.ReplacePathSeparators("test/management-zone")
	err := builder.processConfigSection(m, folderPath)
	assert.NilError(t, err)
}

func TestIsYaml(t *testing.T) {

	assert.Check(t, isYaml("test.yaml"))
	assert.Check(t, isYaml("foo/test.yaml"))
	assert.Check(t, !isYaml("foo/test.json"))
	assert.Check(t, !isYaml(""))
}

func TestStandardizeLocationWithAbsolutePath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := util.ReplacePathSeparators("/general/dashboard/dashboard.json")
	json1 := util.ReplacePathSeparators("/cluster/general/dashboard/dashboard.json")
	standardizedLocation := builder.standardizeLocation(json, "foo")
	standardizedLocation1 := builder.standardizeLocation(json1, "foo")

	expected := util.ReplacePathSeparators("general/dashboard/dashboard.json")
	expected1 := util.ReplacePathSeparators("cluster/general/dashboard/dashboard.json")
	assert.Equal(t, expected, standardizedLocation)
	assert.Equal(t, expected1, standardizedLocation1)
}

func TestStandardizeLocationWithLocalPath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	path := util.ReplacePathSeparators("general/dashboard")
	path1 := util.ReplacePathSeparators("cluster/general/dashboard")
	standardizedLocation := builder.standardizeLocation("dashboard.json", path)
	standardizedLocation1 := builder.standardizeLocation("dashboard.json", path1)

	expected := util.ReplacePathSeparators("general/dashboard/dashboard.json")
	expected1 := util.ReplacePathSeparators("cluster/general/dashboard/dashboard.json")
	assert.Equal(t, expected, standardizedLocation)
	assert.Equal(t, expected1, standardizedLocation1)
}

const projectTestYaml = `
config:
  - dashboard: "my-project-dashboard.json"

dashboard:
  - name: "🦙My Dashboard"
  - value: "Foo"
  - constant: "default value"

dashboard.dev:
  - constant: "overridden in dev"
`

func TestProcessYaml(t *testing.T) {

	factory := config.CreateConfigMockFactory(t)
	fs := util.CreateTestFileSystem()
	err := fs.Mkdir("test/dashboard/", 0777)
	err = afero.WriteFile(fs, "test/dashboard/test-file.yaml", []byte(projectTestYaml), 0664)

	builder := testCreateProjectBuilderWithMock(factory, fs, "testproject", "")

	properties := make(map[string]map[string]string)

	yamlFile := util.ReplacePathSeparators("test/dashboard/test-file.yaml")

	factory.EXPECT().
		NewConfig(fs, "dashboard", "testproject", util.ReplacePathSeparators("test/dashboard/my-project-dashboard.json"), gomock.Any(), testDashboardApi).
		Return(config.GetMockConfig(fs, "my-project-dashboard", "testproject", nil, properties, testDashboardApi, util.ReplacePathSeparators("dashboard/test-file.yaml")), nil)

	err = builder.processYaml(yamlFile)

	assert.NilError(t, err)
	assert.Equal(t, 1, len(builder.configs))

	config := builder.configs[0]
	assert.Check(t, config != nil)
}

var mockGenerateUuidFromConfigIdSuccess = func(projectUniqueId string, configId string) (string, error) {
	return projectUniqueId + "/" + configId, nil
}

var mockGenerateUuidFromConfigIdFail = func(projectUniqueId string, configId string) (string, error) {
	return "", fmt.Errorf("generateUuidFromConfigIdFail")
}

var mockGetRelFilepathSuccess = func(basepath string, targpath string) (string, error) {
	return filepath.Rel(basepath, targpath)
}

var mockGetRelFilepathFail = func(basepath string, targpath string) (string, error) {
	return "", fmt.Errorf("getRelFilepathFail")
}

func TestGenerateConfigUuid(t *testing.T) {
	fullQualifiedProjectFolderName := "/test/folder/env/project"
	projectRootFolder := "/test/folder"

	projectAtTest := &projectImpl{
		id:                       fullQualifiedProjectFolderName,
		projectRootFolder:        projectRootFolder,
		generateUuidFromConfigId: mockGenerateUuidFromConfigIdSuccess,
		getRelFilepath:           mockGetRelFilepathSuccess,
	}

	uuidAtTest, err := projectAtTest.GenerateConfigUuid("my-config-id")
	assert.NilError(t, err)
	assert.Equal(t, "env/project/my-config-id", uuidAtTest)

	projectAtTest.getRelFilepath = mockGetRelFilepathFail

	_, err = projectAtTest.GenerateConfigUuid("my-config-id")
	assert.Error(t, err, "getRelFilepathFail")

	projectAtTest.getRelFilepath = mockGetRelFilepathSuccess
	projectAtTest.generateUuidFromConfigId = mockGenerateUuidFromConfigIdFail

	_, err = projectAtTest.GenerateConfigUuid("my-config-id")
	assert.Error(t, err, "generateUuidFromConfigIdFail")
}
