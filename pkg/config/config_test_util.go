// +build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func CreateConfigMockFactory(t *testing.T) *MockConfigFactory {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	return NewMockConfigFactory(mockCtrl)
}

func GetMockConfig(id string, project string, template util.Template, properties map[string]map[string]string, api api.Api, fileName string) Config {

	return newConfig(id, project, template, properties, api, fileName)
}
