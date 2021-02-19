// +build unit

package yamlcreator

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func CreateYamlCreatorMock(t *testing.T) *MockYamlCreator {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	return NewMockYamlCreator(mockCtrl)
}
