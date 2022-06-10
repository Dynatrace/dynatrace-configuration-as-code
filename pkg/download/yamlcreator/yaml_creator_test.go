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

package yamlcreator

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"gotest.tools/assert"
)

const testEnvironmentName = "test-environment-0"

func TestNewYamlConfig(t *testing.T) {
	config := NewYamlConfig(testEnvironmentName)
	assert.Check(t, config.Detail != nil, "map not initialized")
}

func TestAddConfig(t *testing.T) {
	//test special name in config file
	config := NewYamlConfig(testEnvironmentName)
	config.AddConfig("test", "test 1234")
	assert.Check(t, len(config.Detail["test"]) == 1)
	assert.Check(t, config.Detail["test"][0].Name == "test 1234")
}

// func TestCreateYamlFile(t *testing.T) {
// 	// ctrl := gomock.NewController(t)
// 	config := NewYamlConfig(testEnvironmentName)
// 	config.AddConfig("test", "test 1234")
// 	fileCreator := util.CreateTestFileSystem()
// 	err := config.WriteYamlFile(fileCreator, "", "test")
// 	assert.NilError(t, err)
// }

var mockConfigYaml = []byte(`
config:
- config-id: json-file-name.json

config-id:
- name: Dynatrace entity name
`)

var newMockFs = func(t *testing.T, mockConfigSubPath string, mockApiId string) afero.Fs {
	mockConfigFilePath := filepath.Join(mockConfigSubPath, mockApiId+".yaml")
	mockFs := afero.NewMemMapFs()

	f, err := mockFs.Create(mockConfigFilePath)
	defer f.Close()
	assert.NilError(t, err)

	_, err = f.Write(mockConfigYaml)
	assert.NilError(t, err)

	return mockFs
}

func TestReadYamlFile(t *testing.T) {
	mockConfigSubPath := filepath.Join("config", "sub", "path")
	mockApiId := "api-id"
	mockFs := newMockFs(t, mockConfigSubPath, mockApiId)

	config := NewYamlConfig(testEnvironmentName)

	err := config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.NilError(t, err)

	config.readFile = func(fs afero.Fs, filename string) ([]byte, error) { return []byte{}, fmt.Errorf("readFileFail") }

	err = config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.Error(t, err, "readFileFail")

	config = NewYamlConfig(testEnvironmentName)
	config.unmarshalYaml = func(text, fileName string) (error, map[string]map[string]string) {
		return fmt.Errorf("unmarshalYamlFail"), map[string]map[string]string{}
	}

	err = config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.Error(t, err, "unmarshalYamlFail")

	config = NewYamlConfig(testEnvironmentName)
	config.generateUuidFromConfigId = func(projectUniqueId, configId string) (string, error) {
		return "", fmt.Errorf("generateUuidFromConfigIdFail")
	}

	err = config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.Error(t, err, "generateUuidFromConfigIdFail")
}

func TestUpdateConfig(t *testing.T) {
	mockConfigSubPath := filepath.Join("config", "sub", "path")
	mockApiId := "api-id"
	mockFs := newMockFs(t, mockConfigSubPath, mockApiId)

	config := NewYamlConfig(testEnvironmentName)

	err := config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.NilError(t, err)

	numberConfigs := len(config.Config)
	assert.Equal(t, 1, numberConfigs)

	// Adds config details
	mockEntityId := "new-config-id"
	mockEntityName := "new-config-name"
	mockJsonFileName := "json-file-name.json"

	config.UpdateConfig(mockEntityId, mockEntityName, mockJsonFileName)

	numberConfigs = len(config.Config)
	assert.Equal(t, 2, numberConfigs)

	// Adds config details
	mockEntityId = "another-config-id"
	mockEntityName = "another-config-name"
	mockJsonFileName = "json-file-name.json"

	config.UpdateConfig(mockEntityId, mockEntityName, mockJsonFileName)

	numberConfigs = len(config.Config)
	assert.Equal(t, 3, numberConfigs)

	// Overwrites config details
	mockEntityId = "config-id"
	mockEntityName = "config-name"
	mockJsonFileName = "json-file-name.json"

	config.UpdateConfig(mockEntityId, mockEntityName, mockJsonFileName)

	numberConfigs = len(config.Config)
	assert.Equal(t, 3, numberConfigs)
}

func TestWriteYamlFile(t *testing.T) {
	mockConfigSubPath := filepath.Join("config", "sub", "path")
	mockApiId := "api-id"
	mockFs := newMockFs(t, mockConfigSubPath, mockApiId)

	config := NewYamlConfig(testEnvironmentName)

	err := config.ReadYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.NilError(t, err)

	numberConfigs := len(config.Config)
	assert.Equal(t, 1, numberConfigs)

	// Marks existing config as "downloaded"
	mockEntityId := "config-id"
	mockEntityName := "config-name"
	mockJsonFileName := "json-file-name.json"

	config.UpdateConfig(mockEntityId, mockEntityName, mockJsonFileName)

	numberConfigs = len(config.Config)
	assert.Equal(t, 1, numberConfigs)

	configDetails := config.Detail[mockEntityId]
	assert.Equal(t, true, configDetails[0].IsDownloaded)

	err = config.WriteYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.NilError(t, err)

	config.marshalYaml = func(in interface{}) (out []byte, err error) { return []byte{}, fmt.Errorf("marshalYamlFail") }

	err = config.WriteYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.Error(t, err, "marshalYamlFail")

	config.marshalYaml = yaml.Marshal
	config.writeFile = func(fs afero.Fs, filename string, data []byte, perm fs.FileMode) error {
		return fmt.Errorf("writeFileFail")
	}

	err = config.WriteYamlFile(mockFs, mockConfigSubPath, mockApiId)
	assert.Error(t, err, "writeFileFail")
}
