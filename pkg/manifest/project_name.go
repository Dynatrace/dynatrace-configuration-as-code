// @license
// Copyright 2026 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateProjectName reports whether projectName can be written as a project that lives in a folder of the same name
// inside the output folder, such that the written manifest can be loaded again.
func ValidateProjectName(projectName string) error {
	if strings.ContainsAny(projectName, `/\`) {
		return fmt.Errorf(`project name %q is not allowed to contain '/' or '\'`, projectName)
	}
	if !filepath.IsLocal(projectName) {
		return fmt.Errorf("project name %q must resolve to a folder inside the output folder", projectName)
	}
	return nil
}
