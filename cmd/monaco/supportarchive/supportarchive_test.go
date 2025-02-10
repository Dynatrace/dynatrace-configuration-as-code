/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package supportarchive_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/supportarchive"
)

func TestIsEnabled(t *testing.T) {
	t.Run("New created support-archive is disabled by default", func(t *testing.T) {
		ctx := context.TODO()

		ctx = supportarchive.ContextWithSupportArchive(ctx)

		assert.False(t, supportarchive.IsEnabled(ctx))
	})

	t.Run("Enabled support-archive returns true", func(t *testing.T) {
		ctx := context.TODO()

		ctx = supportarchive.ContextWithSupportArchive(ctx)
		supportarchive.Enable(ctx)

		assert.True(t, supportarchive.IsEnabled(ctx))

	})

	t.Run("Not created support-archive is same as disabled", func(t *testing.T) {
		ctx := context.TODO()

		assert.False(t, supportarchive.IsEnabled(ctx))
	})
}

func TestNewContextWithSupportArchive(t *testing.T) {
	t.Run("support-archive is alreadit created - panic", func(t *testing.T) {
		ctx := context.TODO()

		ctx = supportarchive.ContextWithSupportArchive(ctx)

		assert.Equal(t, ctx, supportarchive.ContextWithSupportArchive(ctx))
	})
}

func TestEnable(t *testing.T) {
	t.Run("If support-archive is already not created - panic", func(t *testing.T) {
		ctx := context.TODO()

		assert.Panics(t, func() {
			supportarchive.Enable(ctx)
		})
	})

	t.Run("enabling support-archive enables it in everywhere", func(t *testing.T) {
		var ctx = context.TODO()
		ctx = supportarchive.ContextWithSupportArchive(ctx)

		ctx1 := context.WithValue(ctx, "first child", "")
		ctx2 := context.WithValue(ctx, "first child", "")

		assert.False(t, supportarchive.IsEnabled(ctx2))
		supportarchive.Enable(ctx1)
		assert.True(t, supportarchive.IsEnabled(ctx2))
	})
}
