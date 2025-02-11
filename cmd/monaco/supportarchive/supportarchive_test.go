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
	t.Run("If support archive is set, it's enabled", func(t *testing.T) {
		ctx := context.TODO()

		ctx = supportarchive.ContextWithSupportArchive(ctx)

		assert.True(t, supportarchive.IsEnabled(ctx))
	})

	t.Run("If support archive isn't set, it's disabled", func(t *testing.T) {
		ctx := context.TODO()

		assert.False(t, supportarchive.IsEnabled(ctx))
	})
}
