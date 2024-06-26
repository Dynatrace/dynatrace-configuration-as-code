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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFeatureFlag(t *testing.T) {
	ff := featureflags.New("MONACO_FEAT_TEST_FLAG", false)

	for _, fv := range []string{"0", "f", "F", "FALSE", "false", "False", "fAlSe", "", "othervalue"} {
		t.Setenv("MONACO_FEAT_TEST_FLAG", fv)
		assert.False(t, ff.Enabled())
	}

	for _, tv := range []string{"1", "t", "T", "TRUE", "true", "tRuE", "True"} {
		t.Setenv("MONACO_FEAT_TEST_FLAG", tv)
		assert.True(t, ff.Enabled())
	}
}

func TestVerifyEnvType(t *testing.T) {
	ff := featureflags.Permanent[featureflags.VerifyEnvironmentType]
	assert.True(t, ff.Enabled())
	t.Setenv(ff.EnvName(), "0")
	assert.False(t, ff.Enabled())
}

func TestDangerousCommands(t *testing.T) {
	ff := featureflags.Permanent[featureflags.DangerousCommands]
	assert.False(t, ff.Enabled())
	t.Setenv(ff.EnvName(), "1")
	assert.True(t, ff.Enabled())
}
