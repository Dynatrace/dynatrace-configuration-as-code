/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package settings

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldDiscard(t *testing.T) {
	t.Run("log-on-grail-activate - should not be persisted when activated false", func(t *testing.T) {
		assert.True(t, defaultSettingsFilters["builtin:logmonitoring.logs-on-grail-activate"].ShouldDiscard(map[string]interface{}{
			"activated": false,
		}))
	})
}

func TestGetFilter(t *testing.T) {
	assert.NotNil(t, Filters{"id": noOpFilter}.Get("id"))
}

func TestNoOpFilterDoesNothing(t *testing.T) {
	assert.False(t, noOpFilter.ShouldDiscard(map[string]interface{}{}))
}
