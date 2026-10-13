//go:build unit

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

package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

func TestValidateEnvironmentName_AcceptsPlainName(t *testing.T) {
	assert.NoError(t, manifest.ValidateEnvironmentName("prod_eu"))
}

func TestValidateEnvironmentName_AcceptsParentFolderName(t *testing.T) {
	assert.NoError(t, manifest.ValidateEnvironmentName(".."))
}

func TestValidateEnvironmentName_RejectsSeparator(t *testing.T) {
	tests := []struct {
		name            string
		environmentName string
		wantError       string
	}{
		{
			name:            "slash",
			environmentName: "prod/eu",
			wantError:       `environment name "prod/eu" is not allowed to contain '/' or '\'`,
		},
		{
			name:            "backslash",
			environmentName: `prod\eu`,
			wantError:       `environment name "prod\\eu" is not allowed to contain '/' or '\'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, manifest.ValidateEnvironmentName(tt.environmentName), tt.wantError)
		})
	}
}
