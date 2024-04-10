/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

func (a API) TestConfiguredApi(t *testing.T) {
	assert.NotEmptyf(t, a.ID, "endpoint %+v have empty ID!", a)
	if a.SingleConfiguration == true {
		assert.Emptyf(t, a.PropertyNameOfGetAllResponse, "endpoint %q have forbiden value combination - when \"SingleConfiguration\" is true, \"PropertyNameOfGetAllResponse\" must be empty! (actual values: %+v)", a.ID, a)
		assert.Falsef(t, a.NonUniqueName, "endpoint %q have forbiden value combination - when \"SingleConfiguration\" is true, \"NonUniqueName\" must be false! (actual values: %+v)", a.ID, a)
	}
}

// noFilter is dummy filter that do nothing.
func NoFilter(a API) bool {
	return noFilter(a)
}
