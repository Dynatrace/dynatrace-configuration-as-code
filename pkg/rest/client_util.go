// +build unit

package rest

import (
	"testing"

	"github.com/golang/mock/gomock"
)

//CreateDynatraceClientMockFactory returns a mock version of the dynatraceclient interface
func CreateDynatraceClientMockFactory(t *testing.T) *MockDynatraceClient {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	return NewMockDynatraceClient(mockCtrl)
}
