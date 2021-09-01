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

//go:build unit
// +build unit

package yamlcreator

import (
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

func TestNewYamlConfig(t *testing.T) {
	config := NewYamlConfig()
	assert.Check(t, config.Detail != nil, "map not initialized")
}

func TestAddConfig(t *testing.T) {
	//test special name in config file
	config := NewYamlConfig()
	config.AddConfig("test", "test 1234")
	assert.Check(t, len(config.Detail["test"]) == 1)
	assert.Check(t, config.Detail["test"][0].Name == "test 1234")
}

func TestCreateYamlFile(t *testing.T) {
	// ctrl := gomock.NewController(t)
	config := NewYamlConfig()
	config.AddConfig("test", "test 1234")
	fileCreator := util.CreateTestFileSystem()
	err := config.CreateYamlFile(fileCreator, "", "test")
	assert.NilError(t, err)
}
