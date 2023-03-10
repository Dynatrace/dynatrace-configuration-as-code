//go:build unit

// @license
// Copyright 2022 Dynatrace LLC
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

package client

import (
	"errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	"testing"
)

var givenJson = []byte{1, 2, 3}
var givenError = errors.New("error")

func TestDecoratedClient_ReadById(t *testing.T) {
	a := api.API{}

	client := NewMockClient(gomock.NewController(t))
	limited := LimitClientParallelRequests(client, 1)

	client.EXPECT().ReadConfigById(a, "id").Return(givenJson, givenError)
	j, e := limited.ReadConfigById(a, "id")

	assert.DeepEqual(t, j, givenJson)
	assert.Equal(t, e, givenError)
}
