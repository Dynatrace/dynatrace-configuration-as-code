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

package environment

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/util"
	"gotest.tools/assert"
	"testing"
)

func TestEnvFlag(t *testing.T) {
	util.SetEnv(t, "Test", "1")
	assert.Equal(t, true, FeatureFlagEnabled("Test"), "Feature Flag - Enabled")
	util.SetEnv(t, "Test", "A")
	assert.Equal(t, true, FeatureFlagEnabled("Test"), "Feature Flag with wrong value")
	util.SetEnv(t, "Test", "0")
	assert.Equal(t, false, FeatureFlagEnabled("Test"), "Feature Flag - Disabled")
	util.UnsetEnv(t, "Test")
	assert.Equal(t, false, FeatureFlagEnabled("Test"), "Feature Flag - Not Set")
}
