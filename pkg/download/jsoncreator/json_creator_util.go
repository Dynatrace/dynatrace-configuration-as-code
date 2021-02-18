// +build unit

package jsoncreator

import (
	"testing"

	"github.com/golang/mock/gomock"
)

func CreateJSONCreatorMock(t *testing.T) *MockJSONCreator {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	return NewMockJSONCreator(mockCtrl)
}
