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

package files

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"
)

func TestCreateFolderVirtual(t *testing.T) {
	fileManager := NewInMemoryFileManager()
	//test with easy path
	folderTest(t, fileManager)
}
func TestCreateFolderDisk(t *testing.T) {
	fileManager := NewDiskFileManager()
	folderTest(t, fileManager)
}
func folderTest(t *testing.T, fileManager FileManager) {
	p, err := fileManager.CreateFolder("./test-files/demo-folder-test-create-folder-feature")
	assert.NilError(t, err)
	assert.Equal(t, p, "./test-files/demo-folder-test-create-folder-feature")
}

func TestCreateFileInMemory(t *testing.T) {
	creator := NewInMemoryFileManager()
	fileCreateTest(t, creator)
}
func fileCreateTest(t *testing.T, fileManager FileManager) {
	data := []byte("{\"test\":\"data\"}")
	name, err := fileManager.CreateFile(data, "./test-files/", "test name 43*&@!1", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431")
	//long name
	name, err = fileManager.CreateFile(data, "./test-files/",
		"test name 43*&@!1 with random detail in name longer that 50 characters would be trim by the function", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431withrandomdetailinnamelongerthat50chara")
}
func TestCreateEmptyFile(t *testing.T) {
	fileManager := NewInMemoryFileManager()
	file, err := fileManager.CreateEmptyFile("demofile.yaml")
	assert.NilError(t, err)
	assert.Check(t, file.Name() == "demofile.yaml")
}

type testReadFile struct {
	Test string
}

func TestReadFile(t *testing.T) {
	fileManager := NewInMemoryFileManager()
	file, err := fileManager.ReadFile("./test-files/sample.json")
	assert.NilError(t, err)
	var demo testReadFile
	err = json.Unmarshal(file, &demo)
	assert.NilError(t, err)
	assert.Check(t, demo.Test == "demo")

}
func TestReadDir(t *testing.T) {
	fileManager := NewInMemoryFileManager()
	dir, err := fileManager.ReadDir("./test-files")
	assert.NilError(t, err)
	var fileExistWhenRead = false
	for _, file := range dir {
		if file.Name() == "sample2.json" {
			fileExistWhenRead = true
		}
	}

	assert.Check(t, fileExistWhenRead == true)

}
