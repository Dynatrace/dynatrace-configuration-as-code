// +build unit

package files

import (
	"testing"

	"gotest.tools/assert"
)

func TestCreateFolderVirtual(t *testing.T) {
	creator := NewInMemoryFileCreator()
	//test with easy path
	folderTest(t, creator)
}
func TestCreateFolderDisk(t *testing.T) {
	creator := NewDiskFileCreator()
	folderTest(t, creator)
}
func folderTest(t *testing.T, creator FileCreator) {
	p, err := creator.CreateFolder("/test")
	assert.NilError(t, err)
	assert.Equal(t, p, "/test")
	//test with complex name
	p, err = creator.CreateFolder("/test 23 a!")
	assert.NilError(t, err)
	assert.Equal(t, p, "/test 23 a!")
}

func TestCreateFileInMemory(t *testing.T) {
	creator := NewInMemoryFileCreator()
	fileCreateTest(t, creator)
}
func fileCreateTest(t *testing.T, creator FileCreator) {
	data := []byte("{\"test\":\"data\"}")
	name, err := creator.CreateFile(data, "../../../cmd/monaco/.logs/", "test name 43*&@!1", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431")
	//long name
	name, err = creator.CreateFile(data, "../../../cmd/monaco/.logs/", "test name 43*&@!1 with random detail in name longer that 50 characters would be trim by the function", ".json")
	assert.NilError(t, err)
	assert.Equal(t, name, "testname431withrandomdetailinnamelongerthat50chara")

}
