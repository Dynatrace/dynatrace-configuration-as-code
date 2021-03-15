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
	p, err := fileManager.CreateFolder("./test")
	assert.NilError(t, err)
	assert.Equal(t, p, "./test")
	//test with complex name
	p, err = fileManager.CreateFolder("./test 23 a!")
	assert.NilError(t, err)
	assert.Equal(t, p, "./test 23 a!")
}

func TestCreateFileInMemory(t *testing.T) {
	creator := NewInMemoryFileManager()
	fileCreateTest(t, creator)
}
func fileCreateTest(t *testing.T, fileManager FileManager) {
	data := []byte("{\"test\":\"data\"}")
	name, err := fileManager.CreateFile(data, "../../../cmd/monaco/.logs/", "test name 43*&@!1", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431")
	//long name
	name, err = fileManager.CreateFile(data, "../../../cmd/monaco/.logs/", "test name 43*&@!1 with random detail in name longer that 50 characters would be trim by the function", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431withrandomdetailinnamelongerthat50chara")

}
