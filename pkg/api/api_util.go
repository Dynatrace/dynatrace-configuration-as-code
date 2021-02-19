// +build unit

package api

import (
	"testing"

	"github.com/golang/mock/gomock"
)

//CreateAPIMockFactory returns a mock version of the api interface
func CreateAPIMockFactory(t *testing.T) *MockApi {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	return NewMockApi(mockCtrl)
}
