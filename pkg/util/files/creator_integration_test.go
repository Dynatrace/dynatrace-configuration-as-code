// +build integration

package files

import (
	"testing"
)

//this test depends on OS
func TestCreateFileDisk(t *testing.T) {
	creator := NewDiskFileCreator()
	fileCreateTest(t, creator)
}
