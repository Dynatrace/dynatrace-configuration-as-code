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

func TestValidateProjectName_AcceptsPlainName(t *testing.T) {
	assert.NoError(t, manifest.ValidateProjectName("project_prod"))
}

func TestValidateProjectName_RejectsSeparator(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		wantError   string
	}{
		{
			name:        "slash",
			projectName: "project_prod/eu",
			wantError:   `project name "project_prod/eu" is not allowed to contain '/' or '\'`,
		},
		{
			name:        "backslash",
			projectName: `project_prod\eu`,
			wantError:   `project name "project_prod\\eu" is not allowed to contain '/' or '\'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, manifest.ValidateProjectName(tt.projectName), tt.wantError)
		})
	}
}

func TestValidateProjectName_RejectsNameOutsideOutputFolder(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		wantError   string
	}{
		{
			name:        "parent folder",
			projectName: "..",
			wantError:   `project name ".." must resolve to a folder inside the output folder`,
		},
		{
			name:        "empty",
			projectName: "",
			wantError:   `project name "" must resolve to a folder inside the output folder`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualError(t, manifest.ValidateProjectName(tt.projectName), tt.wantError)
		})
	}
}
