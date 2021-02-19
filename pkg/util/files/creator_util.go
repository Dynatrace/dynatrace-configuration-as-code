// +build unit

package files

import (
	"testing"

	"github.com/golang/mock/gomock"
)

//CreateFileCreatorMockFactory returns a mock version of the filecreator interface
func CreateFileCreatorMockFactory(t *testing.T) *MockFileCreator {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	return NewMockFileCreator(mockCtrl)
}
