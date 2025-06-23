//go:build unit

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

package featureflags_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
)

func TestFeatureFlag(t *testing.T) {
	ff := featureflags.DangerousCommands // here can be any FF with false as default

	for _, fv := range []string{"0", "f", "F", "FALSE", "false", "False", "fAlSe", "", "othervalue"} {
		t.Setenv(ff.EnvName(), fv)
		assert.False(t, ff.Enabled())
	}

	for _, tv := range []string{"1", "t", "T", "TRUE", "true", "tRuE", "True"} {
		t.Setenv(ff.EnvName(), tv)
		assert.True(t, ff.Enabled())
	}
}

func TestVerifyEnvType(t *testing.T) {
	ff := featureflags.VerifyEnvironmentType
	assert.True(t, ff.Enabled())
	t.Setenv(ff.EnvName(), "0")
	assert.False(t, ff.Enabled())
}

func TestDangerousCommands(t *testing.T) {
	ff := featureflags.DangerousCommands
	assert.False(t, ff.Enabled())
	t.Setenv(ff.EnvName(), "1")
	assert.True(t, ff.Enabled())
}

func TestFeatureFlagID_Enabled(t *testing.T) {
	t.Run("works for temporary environment variables", func(t *testing.T) {
		ff := featureflags.IgnoreSkippedConfigs // any FF from the temporaryDefaultValues

		assert.NotPanics(t, func() {
			ff.Enabled()
		})

		t.Setenv(ff.String(), strconv.FormatBool(true))
		assert.True(t, ff.Enabled(), "feature flag must be enabled")

		t.Setenv(ff.String(), strconv.FormatBool(false))
		assert.False(t, ff.Enabled(), "feature flag must be disabled")
	})

	t.Run("works for permanent environment variables", func(t *testing.T) {
		ff := featureflags.VerifyEnvironmentType // any FF from the permanentDefaultValues

		assert.NotPanics(t, func() {
			ff.Enabled()
		})

		t.Setenv(ff.String(), strconv.FormatBool(true))
		assert.True(t, ff.Enabled(), "feature flag must be enabled")

		t.Setenv(ff.String(), strconv.FormatBool(false))
		assert.False(t, ff.Enabled(), "feature flag must be disabled")
	})

	t.Run("string is not FeatureFlag", func(t *testing.T) {
		assert.Panics(t, func() {
			ff := featureflags.IgnoreSkippedConfigs
			ff = "THIS_IS_UNTYPED_CONST"
			ff.Enabled()
		})
	})
}
