/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAdminAccess(t *testing.T) {
	t.Run("returns true when set to true", func(t *testing.T) {
		ctx := NewContextWithAdminAccess(t.Context(), true)
		assert.True(t, GetAdminAccess(ctx))
	})

	t.Run("returns false when set to false", func(t *testing.T) {
		ctx := NewContextWithAdminAccess(t.Context(), false)
		assert.False(t, GetAdminAccess(ctx))
	})

	t.Run("returns false when not set", func(t *testing.T) {
		assert.False(t, GetAdminAccess(t.Context()))
	})

	t.Run("last value wins", func(t *testing.T) {
		ctx := NewContextWithAdminAccess(t.Context(), false)
		ctx = NewContextWithAdminAccess(ctx, true)
		assert.True(t, GetAdminAccess(ctx))
	})
}
