//go:build unit

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

package v1

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func testCreateProjectBuilder(projectsRoot string) projectBuilder {

	return projectBuilder{
		projectRootFolder: projectsRoot,
		apis:              createTestApis(),
		configProvider: func(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.API) (*Config, error) {
			return &Config{id,
				project,
				properties,
				nil,
				api, fileName,
			}, nil
		},
		configs: make([]*Config, 10),
	}
}

func testCreateProjectBuilderWithMock(configProvider configProvider, fs afero.Fs, projectId string, projectsRoot string) projectBuilder {

	return projectBuilder{
		projectRootFolder: projectsRoot,
		projectId:         projectId,
		apis:              createTestApis(),
		configs:           make([]*Config, 0),
		configProvider:    configProvider,
		fs:                fs,
	}
}

func createTestApis() api.APIs {

	apis := make(api.APIs)
	apis["alerting-profile"] = testAlertingProfileApi
	apis["management-zone"] = testManagementZoneApi
	apis["dashboard"] = testDashboardApi

	return apis
}

var testAlertingProfileApi = api.API{ID: "alerting-profile", URLPath: "/api/config/v1/alertingProfiles"}
var testManagementZoneApi = api.API{ID: "management-zone", URLPath: "/api/config/v1/managementZones"}
var testDashboardApi = api.API{ID: "dashboard", URLPath: "/api/config/v1/dashboards", NonUniqueName: true, DeprecatedBy: "dashboard-v2"}

func TestGetPathSuccess(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := files.ReplacePathSeparators("management-zone/testytest.json")

	err, configType := builder.getConfigTypeFromLocation(json)

	assert.NoError(t, err)
	assert.Equal(t, "management-zone", configType.ID)

	err, configType = builder.getConfigTypeFromLocation(json)

	assert.NoError(t, err)
	assert.Equal(t, "management-zone", configType.ID)
}

func TestGetPathTooLittleArgs(t *testing.T) {

	builder := testCreateProjectBuilder("")
	err, _ := builder.getConfigTypeFromLocation("testytest.json")

	assert.Error(t, err, "path testytest.json too short")
}

func TestRemoveYamlFromPath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	yaml := files.ReplacePathSeparators("project/dashboards/config.yaml")
	expected := files.ReplacePathSeparators("project/dashboards")
	err, result := builder.removeYamlFileFromPath(yaml)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestRemoveYamlFromPathWhenPathIsTooShort(t *testing.T) {

	builder := testCreateProjectBuilder("")
	err, _ := builder.removeYamlFileFromPath("config.yaml")

	assert.Error(t, err, "path config.yaml too short")
}

func TestGetApiInformationFromLocation(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := files.ReplacePathSeparators("test/management-zone/testytest.json")
	err, apiInfo := builder.getExtendedInformationFromLocation(json)

	assert.NoError(t, err)
	assert.Equal(t, testManagementZoneApi, apiInfo)
}

func TestGetConfigTypeInformationFromLocation(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := files.ReplacePathSeparators("test/alerting-profile/testytest.json")
	json1 := files.ReplacePathSeparators("cluster/test/alerting-profile/testytest.json")
	json2 := files.ReplacePathSeparators("config/cluster/test/alerting-profile/testytest.json")
	err, api := builder.getExtendedInformationFromLocation(json)
	err1, api1 := builder.getExtendedInformationFromLocation(json1)
	err2, api2 := builder.getExtendedInformationFromLocation(json2)

	assert.NoError(t, err)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "alerting-profile", api.ID)
	assert.Equal(t, "alerting-profile", api1.ID)
	assert.Equal(t, "alerting-profile", api2.ID)
}

func TestGetApiFromLocationApiNotFound(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := files.ReplacePathSeparators("test/notexisting/testytest.json")
	err, _ := builder.getExtendedInformationFromLocation(json)

	assert.ErrorContains(t, err, "API was unknown")
}

func TestProcessConfigSection(t *testing.T) {

	fs := testutils.CreateTestFileSystem()
	builder := testCreateProjectBuilderWithMock(func(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.API) (*Config, error) {
		return &Config{id, project, properties, nil, api, fileName}, nil
	}, fs, "testProject", "")

	m := make(map[string]map[string]string)

	m["config"] = make(map[string]string)

	m["config"]["test1"] = files.ReplacePathSeparators("/test/management-zone/zoneA.json")
	m["config"]["test2"] = files.ReplacePathSeparators("/test/alerting-profile/profile.json")

	folderPath := files.ReplacePathSeparators("test/management-zone")
	err := builder.processConfigSection(m, folderPath)
	assert.NoError(t, err)
}

func TestProcessConfigSectionWithProjectRootParameter(t *testing.T) {

	fileReaderMock := testutils.CreateTestFileSystem()
	builder := testCreateProjectBuilderWithMock(func(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.API) (*Config, error) {
		return &Config{id,
			project,
			properties,
			nil,
			api,
			fileName,
		}, nil
	}, fileReaderMock, "test", "testProjectsRoot")

	m := make(map[string]map[string]string)

	m["config"] = make(map[string]string)

	m["config"]["testconfig1"] = files.ReplacePathSeparators("/test/management-zone/zoneA.json")
	m["config"]["testconfig2"] = files.ReplacePathSeparators("/test/alerting-profile/profile.json")

	folderPath := files.ReplacePathSeparators("test/management-zone")
	err := builder.processConfigSection(m, folderPath)
	assert.NoError(t, err)
}

func TestStandardizeLocationWithAbsolutePath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	json := files.ReplacePathSeparators("/general/dashboard/dashboard.json")
	json1 := files.ReplacePathSeparators("/cluster/general/dashboard/dashboard.json")
	standardizedLocation := builder.standardizeLocation(json, "foo")
	standardizedLocation1 := builder.standardizeLocation(json1, "foo")

	expected := files.ReplacePathSeparators("/general/dashboard/dashboard.json")
	expected1 := files.ReplacePathSeparators("/cluster/general/dashboard/dashboard.json")
	assert.Equal(t, expected, standardizedLocation)
	assert.Equal(t, expected1, standardizedLocation1)
}

func TestStandardizeLocationWithLocalPath(t *testing.T) {

	builder := testCreateProjectBuilder("")
	path := files.ReplacePathSeparators("general/dashboard")
	path1 := files.ReplacePathSeparators("cluster/general/dashboard")
	standardizedLocation := builder.standardizeLocation("dashboard.json", path)
	standardizedLocation1 := builder.standardizeLocation("dashboard.json", path1)

	expected := files.ReplacePathSeparators("general/dashboard/dashboard.json")
	expected1 := files.ReplacePathSeparators("cluster/general/dashboard/dashboard.json")
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

	fs := testutils.CreateTestFileSystem()
	err := fs.Mkdir("test/dashboard/", 0777)
	err = afero.WriteFile(fs, "test/dashboard/test-file.yaml", []byte(projectTestYaml), 0664)

	builder := testCreateProjectBuilderWithMock(func(fs afero.Fs, id string, project string, fileName string, properties map[string]map[string]string, api api.API) (*Config, error) {
		return &Config{id,
			project,
			properties,
			nil,
			api,
			fileName,
		}, nil
	}, fs, "testproject", "")

	yamlFile := files.ReplacePathSeparators("test/dashboard/test-file.yaml")

	err = builder.processYaml(yamlFile, template.UnmarshalYaml)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(builder.configs))

	config := builder.configs[0]
	assert.NotNil(t, config)
}

func Test_projectBuilder_resolveDuplicateIDs(t *testing.T) {
	tests := []struct {
		name         string
		givenConfigs []*Config
		wantConfigs  []*Config
	}{
		{
			"Works for empty configs",
			[]*Config{},
			[]*Config{},
		},
		{
			"Works for no duplicates",
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-c",
					properties: map[string]map[string]string{
						"config-c": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-d",
					properties: map[string]map[string]string{
						"config-d": {
							"some-prop": "val",
						},
					},
				},
			},
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-c",
					properties: map[string]map[string]string{
						"config-c": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-d",
					properties: map[string]map[string]string{
						"config-d": {
							"some-prop": "val",
						},
					},
				},
			},
		},
		{
			"Renames duplicates",
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
			},
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a-1",
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b-1",
					properties: map[string]map[string]string{
						"config-b-1": {
							"some-prop": "val",
						},
					},
				},
			},
		},
		{
			"Renames several duplicates",
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
			},
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a-1",
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-a-2",
					properties: map[string]map[string]string{
						"config-a-2": {
							"some-prop": "val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
					},
				},
			},
		},
		{
			"Does not rename overlaps with different API types",
			[]*Config{
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
			},
			[]*Config{
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
			},
		},
		{
			"Does keep a separate counter per API",
			[]*Config{
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
			},
			[]*Config{
				{
					id:  "config-a",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a-1",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a-2",
					api: testDashboardApi,
					properties: map[string]map[string]string{
						"config-a-2": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a-1",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
					},
				},
				{
					id:  "config-a-2",
					api: testManagementZoneApi,
					properties: map[string]map[string]string{
						"config-a-2": {
							"some-prop": "val",
						},
					},
				},
			},
		},
		{
			"Renames override properties of duplicates",
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a.prod": {
							"some-prop": "Special PROD val",
						},
					},
				},
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a.prod": {
							"some-prop": "Different PROD val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
						"config-b.staging": {
							"some-prop": "Special STAGING val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
						"config-b.staging": {
							"some-prop": "Different STAGING val",
						},
					},
				},
			},
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a.prod": {
							"some-prop": "Special PROD val",
						},
					},
				},
				{
					id: "config-a-1",
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
						"config-a-1.prod": {
							"some-prop": "Different PROD val",
						},
					},
				},
				{
					id: "config-b",
					properties: map[string]map[string]string{
						"config-b": {
							"some-prop": "val",
						},
						"config-b.staging": {
							"some-prop": "Special STAGING val",
						},
					},
				},
				{
					id: "config-b-1",
					properties: map[string]map[string]string{
						"config-b-1": {
							"some-prop": "val",
						},
						"config-b-1.staging": {
							"some-prop": "Different STAGING val",
						},
					},
				},
			},
		},
		{
			"Does not rename properties of duplicates that just partially match ID",
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a-something-else": {
							"some-prop": "Just another config",
						},
					},
				},
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a-something-else": {
							"some-prop": "Just another config",
						},
					},
				},
			},
			[]*Config{
				{
					id: "config-a",
					properties: map[string]map[string]string{
						"config-a": {
							"some-prop": "val",
						},
						"config-a-something-else": {
							"some-prop": "Just another config",
						},
					},
				},
				{
					id: "config-a-1",
					properties: map[string]map[string]string{
						"config-a-1": {
							"some-prop": "val",
						},
						"config-a-something-else": {
							"some-prop": "Just another config",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &projectBuilder{
				configs: tt.givenConfigs,
			}
			p.resolveDuplicateIDs()

			assert.EqualValues(t, tt.wantConfigs, tt.givenConfigs)
		})
	}
}
