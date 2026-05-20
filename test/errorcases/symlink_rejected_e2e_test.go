//go:build integration

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

package errorcases

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
)

// TestSymlinkedTemplate_IsRejected verifies that a template file which is itself a symlink
// is rejected during config loading, even with --dry-run (no live environment needed).
func TestSymlinkedTemplate_IsRejected(t *testing.T) {
	t.Setenv("SOME_TOKEN", "dummy")
	manifest := "testdata/configs-with-symlinks/manifest-symlinked-template.yaml"

	logOutput := strings.Builder{}
	cmd, _ := runner.BuildCmdWithLogSpy(afero.NewOsFs(), &logOutput)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")
	assert.Contains(t, strings.ToLower(logOutput.String()), "symbolic link")
}

// TestSymlinkedGrandparentDirectory_IsRejected verifies that a symlink deeper in the path
// (grandparent, not the immediate parent) is also caught. The template path is
// real-dir/data/nested/template.json where real-dir/ is a real directory but data/ inside
// it is a symlink.
func TestSymlinkedGrandparentDirectory_IsRejected(t *testing.T) {
	t.Setenv("SOME_TOKEN", "dummy")
	manifest := "testdata/configs-with-symlinks/manifest-symlinked-grandparent.yaml"

	logOutput := strings.Builder{}
	cmd, _ := runner.BuildCmdWithLogSpy(afero.NewOsFs(), &logOutput)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")
	assert.Contains(t, strings.ToLower(logOutput.String()), "symbolic link")
}

// a symlinked parent directory is rejected. This is the parent-directory bypass: before the
// fix, lstat only checked the final path component and would transparently follow a symlinked
// parent directory.
func TestSymlinkedParentDirectory_IsRejected(t *testing.T) {
	t.Setenv("SOME_TOKEN", "dummy")
	manifest := "testdata/configs-with-symlinks/manifest-symlinked-parent.yaml"

	logOutput := strings.Builder{}
	cmd, _ := runner.BuildCmdWithLogSpy(afero.NewOsFs(), &logOutput)
	cmd.SetArgs([]string{"deploy", "--verbose", "--dry-run", manifest})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")
	assert.Contains(t, strings.ToLower(logOutput.String()), "symbolic link")
}
