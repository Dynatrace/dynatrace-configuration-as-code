//go:build unit

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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUrl(t *testing.T) {
	assert.Equal(t, "https://url/to/dev/environment/api/config/v1/managementZones", (&API{apiPath: "/api/config/v1/managementZones"}).GetUrl(testDevEnvironment.GetEnvironmentUrl()))
}

func Test_newAPI(t *testing.T) {
	template := &API{apiPath: "path_to_heaven"}
	actual := newAPI("newID", template)
	assert.NotSame(t, template, actual, "references must be different")

	basicAPI := newAPI("name", &API{apiPath: "path_to_heaven"})
	assert.Equal(t, API{id: "name", apiPath: "path_to_heaven", propertyNameOfGetAllResponse: standardApiPropertyNameOfGetAllResponse}, *basicAPI)
}
