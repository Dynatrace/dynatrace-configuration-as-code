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

package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/delete"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"

	"gotest.tools/assert"
)

const mockWorkdir = "/"
const mockEnvironmentsFilePath = "/environments.yaml"
const mockSpecificEnvironment = ""
const mockProj = ""
const mockContinueOnError = false

var mockConfigYamlFileContent = []byte(`
---
config:
  - uuid: config.json

uuid:
- name: Instance name
`)

var mockConfigSkipDeploymentYamlFileContent = []byte(`
---
config:
  - uuid: config.json
  - uuid2: config.json

uuid:
- name: Instance name

uuid2:
- name: Another instance name
- skipDeployment: "true"
`)

var mockDuplicateConfigYamlFileContent = []byte(`
---
config:
- uuid1: config.json
- uuid2: config.json

uuid1:
- name: Duplicate name

uuid2:
- name: Duplicate name
`)

var mockDuplicateAcrossEnvironmentsConfigYamlFileContent = []byte(`
---
config:
- uuid1: config.json

uuid1.dev:
- name: Duplicate name

uuid2.prod:
- name: Duplicate name
`)

var mockConfigJsonFileContent = []byte(`
{
	"name": "{{.name}}"
}
`)

var mockDeleteFileContent = []byte(`
---
delete:
  - "alerting-profile/Star Trek Service"
  - "dashboard/Alpha Quadrant"
`)

// TBD: This could be properly mocked
var mockLoadProjectsToDeploy = project.LoadProjectsToDeploy
var mockLoadConfigsToDelete = delete.LoadConfigsToDelete
var mockLoadEnvironmentList = func(specificEnvironment, environmentsFilePath string, fs afero.Fs) (environments map[string]environment.Environment, errorList []error) {
	os.Setenv("DEV", "DEV_API_TOKEN")
	os.Setenv("PROD", "PROD_API_TOKEN")

	environments = map[string]environment.Environment{}
	environments["dev"] = environment.NewEnvironment("dev", "Dev", "", "https://url/to/dev/environment", "DEV")
	environments["prod"] = environment.NewEnvironment("prod", "Prod", "", "https://url/to/prod/environment", "PROD")

	return environments, []error{}
}

var testGetExecuteApis = func() map[string]api.Api {
	apis := make(map[string]api.Api)
	apis["calculated-metrics-log"] = api.NewStandardApi("calculated-metrics-log", "/api", false)
	apis["alerting-profile"] = api.NewStandardApi("alerting-profile", "/api", false)
	apis["dashboard"] = api.NewStandardApi("dashboard", "/api", true)
	return apis
}

var mockNewHandler = func(t *testing.T) *Deploy {
	mockFs := afero.NewMemMapFs()

	// delete.yaml
	f, err := mockFs.Create("/delete.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockDeleteFileContent)
	assert.NilError(t, err)

	// projects
	mockFs.MkdirAll("/projectA/calculated-metrics-log", 0777)
	f, err = mockFs.Create("/projectA/calculated-metrics-log/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigSkipDeploymentYamlFileContent)
	assert.NilError(t, err)
	f, err = mockFs.Create("/projectA/calculated-metrics-log/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	mockFs.MkdirAll("/projectA/alerting-profile", 0777)
	f, err = mockFs.Create("/projectA/alerting-profile/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigYamlFileContent)
	assert.NilError(t, err)
	f, err = mockFs.Create("/projectA/alerting-profile/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	mockFs.MkdirAll("/projectB/dashboard", 0777)
	f, err = mockFs.Create("/projectB/dashboard/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigYamlFileContent)
	assert.NilError(t, err)
	f, err = mockFs.Create("/projectB/dashboard/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	mockApis := testGetExecuteApis()

	deployHandler := &Deploy{
		fs:                   mockFs,
		workingDir:           filepath.Clean(mockWorkdir),
		environmentsFilePath: mockEnvironmentsFilePath,
		loadEnvironmentList:  mockLoadEnvironmentList,
		loadProjectsToDeploy: mockLoadProjectsToDeploy,
		loadConfigsToDelete:  mockLoadConfigsToDelete,
		failOnError:          func(err error, msg string) { return },
		apis:                 mockApis,
	}

	return deployHandler
}

func TestFailsOnMissingFileName(t *testing.T) {
	_, err := environment.LoadEnvironmentList("dev", "", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 1, "Expected error return")
}

func TestLoadsEnvironmentListCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 3, "Expected to load test environments 1-3!")
}

func TestLoadSpecificEnvironmentCorrectly(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test2", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 0, "Expected no error")
	assert.Assert(t, len(environments) == 1, "Expected to load test environment 2 only!")
	assert.Assert(t, environments["test2"] != nil, "test2 environment not found in returned list!")
}

func TestMissingSpecificEnvironmentResultsInError(t *testing.T) {
	environments, err := environment.LoadEnvironmentList("test42", "../../cmd/monaco/test-resources/test-environments.yaml", util.CreateTestFileSystem())
	assert.Assert(t, len(err) == 1, "Expected error from referencing unknown environment")
	assert.Assert(t, len(environments) == 0, "Expected to get empty environment map even on error")
}

func TestNewHandler(t *testing.T) {
	mockFs := afero.NewMemMapFs()

	deployHandler, err := NewHandler(mockWorkdir, mockFs, mockEnvironmentsFilePath)
	assert.NilError(t, err)
	assert.Equal(t, mockWorkdir, deployHandler.workingDir)
	assert.Equal(t, mockEnvironmentsFilePath, deployHandler.environmentsFilePath)
}

func TestDeploy(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	deployHandler := mockNewHandler(t)

	mockDynatraceClient := rest.NewMockDynatraceClient(mockCtrl)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	mockDynatraceClient.
		EXPECT().
		UpsertByEntityId(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(api.DynatraceEntity{}, nil).
		AnyTimes()

	mockDynatraceClient.
		EXPECT().
		UpsertByName(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(api.DynatraceEntity{}, nil).
		AnyTimes()

	err := deployHandler.Deploy(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.NilError(t, err)

	// Test loadEnvironmentList fail
	deployHandler.loadEnvironmentList = func(specificEnvironment, environmentsFilePath string, fs afero.Fs) (environments map[string]environment.Environment, errorList []error) {
		return map[string]environment.Environment{}, []error{fmt.Errorf("loadEnvironmentListFail1"), fmt.Errorf("loadEnvironmentListFail2")}
	}

	err = deployHandler.Deploy(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.ErrorContains(t, err, "errors during deployment")

	// Test loadProjectsToDeploy fail
	failOnErrorCount := 0
	deployHandler = mockNewHandler(t)
	deployHandler.loadProjectsToDeploy = func(fs afero.Fs, specificProjectToDeploy string, apis map[string]api.Api, path string) (projectsToDeploy []project.Project, err error) {
		return []project.Project{}, fmt.Errorf("loadProjectsToDeployFail")
	}
	deployHandler.failOnError = func(err error, msg string) {
		failOnErrorCount++
		return
	}
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	err = deployHandler.Deploy(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.Equal(t, 1, failOnErrorCount)

	// Test execute fail
	deployHandler = mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, fmt.Errorf("executeFail")
	}

	err = deployHandler.Deploy(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.ErrorContains(t, err, "errors during deployment")

	err = deployHandler.Deploy(mockSpecificEnvironment, mockProj, true)
	assert.ErrorContains(t, err, "errors during deployment")
}

func TestDeployDryRun(t *testing.T) {
	deployHandler := mockNewHandler(t)

	err := deployHandler.DeployDryRun(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.NilError(t, err)

	// Test loadEnvironmentList fail
	deployHandler.loadEnvironmentList = func(specificEnvironment, environmentsFilePath string, fs afero.Fs) (environments map[string]environment.Environment, errorList []error) {
		return map[string]environment.Environment{}, []error{fmt.Errorf("loadEnvironmentListFail1"), fmt.Errorf("loadEnvironmentListFail2")}
	}

	err = deployHandler.DeployDryRun(mockSpecificEnvironment, mockProj, mockContinueOnError)
	assert.ErrorContains(t, err, "errors during validation")
}

func TestDelete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockDynatraceClient := rest.NewMockDynatraceClient(mockCtrl)

	deployHandler := mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	mockDynatraceClient.
		EXPECT().
		DeleteByName(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	err := deployHandler.Delete(mockSpecificEnvironment, mockProj)
	assert.NilError(t, err)

	// Test DeleteByName fail
	mockDynatraceClient = rest.NewMockDynatraceClient(mockCtrl)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	mockDynatraceClient.
		EXPECT().
		DeleteByName(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("DeleteByNameFail")).
		Times(1)

	err = deployHandler.Delete(mockSpecificEnvironment, mockProj)
	assert.Error(t, err, "DeleteByNameFail")

	// Test newDynatraceClient fail
	mockDynatraceClient = rest.NewMockDynatraceClient(mockCtrl)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, fmt.Errorf("newDynatraceClientFail")
	}

	mockDynatraceClient.
		EXPECT().
		DeleteByName(gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("DeleteByNameFail")).
		Times(0)

	err = deployHandler.Delete(mockSpecificEnvironment, mockProj)
	assert.Error(t, err, "newDynatraceClientFail")
}

func TestDeleteDryRun(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockDynatraceClient := rest.NewMockDynatraceClient(mockCtrl)

	deployHandler := mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	mockDynatraceClient.
		EXPECT().
		DeleteByName(gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	err := deployHandler.DeleteDryRun(mockSpecificEnvironment, mockProj)
	assert.NilError(t, err)
}

func TestExecuteWithDuplicateConfigNames(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockDynatraceClient := rest.NewMockDynatraceClient(mockCtrl)

	mockDynatraceClient.
		EXPECT().
		UpsertByEntityId(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(api.DynatraceEntity{}, nil).
		AnyTimes()

	mockDynatraceClient.
		EXPECT().
		UpsertByName(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(api.DynatraceEntity{}, nil).
		AnyTimes()

	// Create duplicate config within same project
	deployHandler := mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	err := deployHandler.fs.MkdirAll("/projectC/calculated-metrics-log", 0777)
	assert.NilError(t, err)
	f, err := deployHandler.fs.Create("/projectC/calculated-metrics-log/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockDuplicateConfigYamlFileContent)
	assert.NilError(t, err)
	f, err = deployHandler.fs.Create("/projectC/calculated-metrics-log/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	environments, _ := mockLoadEnvironmentList("", "", deployHandler.fs)
	apis := testGetExecuteApis()
	projects, _ := mockLoadProjectsToDeploy(deployHandler.fs, "", apis, deployHandler.workingDir)

	deployErrors := deployHandler.execute(environments["dev"], projects, false, false)
	assert.Equal(t, 1, len(deployErrors))
	assert.ErrorContains(t, deployErrors[0], "duplicate UID 'calculated-metrics-log/Duplicate name'")

	// Create duplicate config in different projects
	deployHandler = mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	err = deployHandler.fs.MkdirAll("/projectB/calculated-metrics-log", 0777)
	assert.NilError(t, err)
	f, err = deployHandler.fs.Create("/projectB/calculated-metrics-log/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigYamlFileContent)
	assert.NilError(t, err)
	f, err = deployHandler.fs.Create("/projectB/calculated-metrics-log/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	environments, _ = mockLoadEnvironmentList("", "", deployHandler.fs)
	apis = testGetExecuteApis()
	projects, _ = mockLoadProjectsToDeploy(deployHandler.fs, "", apis, deployHandler.workingDir)

	deployErrors = deployHandler.execute(environments["dev"], projects, false, false)
	assert.Equal(t, 1, len(deployErrors))
	assert.ErrorContains(t, deployErrors[0], "duplicate UID 'calculated-metrics-log/Instance name'")

	// Create duplicate config across environments
	deployHandler = mockNewHandler(t)
	deployHandler.newDynatraceClient = func(environmentUrl, token string) (rest.DynatraceClient, error) {
		return mockDynatraceClient, nil
	}

	err = deployHandler.fs.MkdirAll("/projectB/calculated-metrics-log", 0777)
	assert.NilError(t, err)
	f, err = deployHandler.fs.Create("/projectB/calculated-metrics-log/config.yaml")
	assert.NilError(t, err)
	_, err = f.Write(mockDuplicateAcrossEnvironmentsConfigYamlFileContent)
	assert.NilError(t, err)
	f, err = deployHandler.fs.Create("/projectB/calculated-metrics-log/config.json")
	assert.NilError(t, err)
	_, err = f.Write(mockConfigJsonFileContent)
	assert.NilError(t, err)

	environments, _ = mockLoadEnvironmentList("", "", deployHandler.fs)
	apis = testGetExecuteApis()
	projects, _ = mockLoadProjectsToDeploy(deployHandler.fs, "", apis, deployHandler.workingDir)

	deployErrors = deployHandler.execute(environments["dev"], projects, false, false)
	assert.Equal(t, 0, len(deployErrors))
}

// TODO (CDF-6511) Currently here UnmarshallYaml logs fatal, only ever returns nil errors!
// func TestInvalidEnvironmentFileResultsInError(t *testing.T) {
// 	_, err := environment.LoadEnvironmentList("", "test-resources/invalid-environmentsfile.yaml")
// 	assert.Assert(t, err != nil, "Expected error return")
// }

// TODO (CDF-6511) add tests when execute failures of single environments don't crash program anymore
