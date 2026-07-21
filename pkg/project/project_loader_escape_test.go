//go:build unit

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

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

// TestLoadProjects_BasePathFsEscape_SiblingWithSharedPrefix guards against a
// path-traversal hole in Monaco's project-loading sandbox.
//
// LoadProjects wraps the source filesystem with afero.BasePathFs and afero v1.15.0's
// BasePathFs.RealPath checks containment via strings.HasPrefix, which is not
// path-aware: given workingDir ".../repo", a project Path of "../repo-secrets"
// resolves to ".../repo-secrets", and the naive prefix check accepts it
// because that string starts with ".../repo".
//
// Without an explicit local-path check, a manifest whose project entry uses
// `path: ../repo-secrets` (with working directory `repo/`) would load YAML
// configs from the sibling `repo-secrets/` directory. This test asserts the
// project loader rejects that path outright.
func TestLoadProjects_BasePathFsEscape_SiblingWithSharedPrefix(t *testing.T) {
	// Layout:
	//   <tmp>/
	//     repo/            <- WorkingDir (sandbox root)
	//     repo-secrets/    <- sibling; MUST be unreachable from repo/
	//       leaked/
	//         leaked.yaml  <- config the sandbox should NOT allow loading
	//         leaked.json  <- template referenced by the config
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	workingDir := filepath.Join(tmp, "repo")
	sibling := filepath.Join(tmp, "repo-secrets")
	leakedDir := filepath.Join(sibling, "leaked")
	require.NoError(t, os.MkdirAll(workingDir, 0755))
	require.NoError(t, os.MkdirAll(leakedDir, 0755))

	yaml := `configs:
			- id: leaked
			  config:
				name: leaked
				template: leaked.json
			  type:
				api: dashboard
			`
	require.NoError(t, os.WriteFile(filepath.Join(leakedDir, "leaked.yaml"), []byte(yaml), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(leakedDir, "leaked.json"), []byte("{}"), 0644))

	loaderContext := ProjectLoaderContext{
		KnownApis:  map[string]struct{}{"dashboard": {}},
		WorkingDir: workingDir,
		Manifest: manifest.Manifest{
			Projects: manifest.ProjectDefinitionByProjectID{
				"escape": {
					Name: "escape",
					Path: "../repo-secrets",
				},
			},
			Environments: manifest.Environments{
				SelectedEnvironments: manifest.EnvironmentDefinitionsByName{
					"env": {
						Name: "env",
						Auth: manifest.Auth{
							AccessToken: &manifest.AuthSecret{Name: "env_VAR"},
						},
					},
				},
				AllEnvironmentNames: map[string]struct{}{"env": {}},
			},
		},
		ParametersSerde: config.DefaultParameterParsers,
	}

	got, gotErrs := LoadProjects(t.Context(), afero.NewOsFs(), loaderContext, nil)

	require.NotEmpty(t, gotErrs, "LoadProjects must reject '../repo-secrets' — it escapes workingDir")
	assert.Empty(t, got, "no project should be loaded when the path escapes the sandbox")

	var msgs strings.Builder
	for _, e := range gotErrs {
		msgs.WriteString(e.Error() + "\n")
	}
	assert.Containsf(t, msgs.String(), "escape",
		"error should indicate the path escapes the manifest directory; got: %s", msgs.String())

	// Sanity check: the sibling was actually present on disk, so a passing
	// test genuinely means the loader refused it (not that the file was missing).
	_, statErr := os.Stat(filepath.Join(sibling, "leaked", "leaked.yaml"))
	require.NoError(t, statErr, "sibling fixture should exist for this test to be meaningful")
}
