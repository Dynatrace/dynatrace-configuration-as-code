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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_removeURLsFromPublicAccess(t *testing.T) {
	t.Run("removes URLs", func(t *testing.T) {
		m := map[string]any{
			"publicAccess": map[string]any{
				"urls": []string{"https://some1.dynatrace.com", "https://some2.dynatrace.com", "https://some3.dynatrace.com"},
			},
		}
		removeURLsFromPublicAccess(m)
		require.Contains(t, m, "publicAccess")
		assert.NotContains(t, m["publicAccess"], "urls")
	})

	t.Run("preserves management zones", func(t *testing.T) {
		m := map[string]any{
			"publicAccess": map[string]any{
				"managementZones": []string{"1", "2", "3"},
				"urls":            []string{"https://some1.dynatrace.com", "https://some2.dynatrace.com", "https://some3.dynatrace.com"},
			},
		}
		removeURLsFromPublicAccess(m)
		require.Contains(t, m, "publicAccess")
		assert.Contains(t, m["publicAccess"], "managementZones")
		assert.NotContains(t, m["publicAccess"], "urls")
	})

	t.Run("does not tweak unexpected input", func(t *testing.T) {
		m := map[string]any{
			"otherField": "value",
			"publicAccess": map[string]any{
				"anotherField":    1,
				"managementZones": []string{"1", "2", "3"},
				"urls":            []string{"https://some1.dynatrace.com", "https://some2.dynatrace.com", "https://some3.dynatrace.com"},
			},
		}
		removeURLsFromPublicAccess(m)
		assert.Contains(t, m, "otherField")
		require.Contains(t, m, "publicAccess")
		assert.NotContains(t, m["publicAccess"], "urls")
		assert.Contains(t, m["publicAccess"], "managementZones")
		assert.Contains(t, m["publicAccess"], "anotherField")
	})
}
