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

package deletefile

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v2"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/delete/persistence"
)

const testManifest = `manifestVersion: 1.0
projects:
- name: project-a
- name: project-b
environmentGroups:
- name: default
  environments:
  - name: env1
    url:
      value: http://www.url.com
    auth:
      token:
        name: TOKEN
`

const projectAConfig = `configs:
- id: profile
  config:
    name: Profile A
    template: template.json
  type:
    api: alerting-profile
- id: dashboard
  config:
    name: Dashboard A
    template: template.json
  type:
    api: dashboard
- id: dashboard-share-settings
  config:
    name: Dashboard A
    template: template.json
  type:
    api:
      name: dashboard-share-settings
      scope:
        configId: dashboard
        configType: dashboard
        property: id
        type: reference
`

const projectBConfig = `configs:
- id: profile
  config:
    name: Profile B
    template: template.json
  type:
    api: alerting-profile
`

// setupTestFs writes a manifest with two projects into an in-memory filesystem and returns the fs and the manifest path.
func setupTestFs(t *testing.T) (afero.Fs, string) {
	t.Helper()

	fs := afero.NewMemMapFs()
	baseDir := t.TempDir()

	manifestPath := filepath.Join(baseDir, "manifest.yaml")
	require.NoError(t, afero.WriteFile(fs, manifestPath, []byte(testManifest), 0644))

	writeProject := func(name, cfg string) {
		dir := filepath.Join(baseDir, name, "configs")
		require.NoError(t, fs.MkdirAll(dir, 0755))
		require.NoError(t, afero.WriteFile(fs, filepath.Join(dir, "config.yaml"), []byte(cfg), 0644))
		require.NoError(t, afero.WriteFile(fs, filepath.Join(dir, "template.json"), []byte(`{}`), 0644))
	}
	writeProject("project-a", projectAConfig)
	writeProject("project-b", projectBConfig)

	return fs, manifestPath
}

func readDeleteEntries(t *testing.T, fs afero.Fs, path string) []persistence.DeleteEntry {
	t.Helper()

	content, err := afero.ReadFile(fs, path)
	require.NoError(t, err)

	var def persistence.FullFileDefinition
	require.NoError(t, yaml.Unmarshal(content, &def))
	return def.DeleteEntries
}

func configNamesOfType(entries []persistence.DeleteEntry, cfgType string) []string {
	var names []string
	for _, e := range entries {
		if e.Type == cfgType {
			names = append(names, e.ConfigName)
		}
	}
	return names
}

func TestCommand_FlagDefaults(t *testing.T) {
	cmd := Command(afero.NewMemMapFs())

	tests := []struct {
		flag         string
		shorthand    string
		defaultValue string
	}{
		{flag: "output-folder", shorthand: "o", defaultValue: ""},
		{flag: "file", defaultValue: "delete.yaml"},
		{flag: "project", shorthand: "p", defaultValue: "[]"},
		{flag: "environment", shorthand: "e", defaultValue: "[]"},
		{flag: "types", defaultValue: "[]"},
		{flag: "exclude-types", defaultValue: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flag)
			require.NotNil(t, f, "flag %q must be defined", tt.flag)
			assert.Equal(t, tt.shorthand, f.Shorthand)
			assert.Equal(t, tt.defaultValue, f.DefValue)
		})
	}
}

func TestCommand_RejectsNonYamlManifest(t *testing.T) {
	fs := afero.NewMemMapFs()

	cmd := Command(fs)
	cmd.SetArgs([]string{"manifest.json"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "Expected a .yaml file, but got manifest.json")
}

func TestCommand_FailsIfManifestCannotBeLoaded(t *testing.T) {
	fs := afero.NewMemMapFs()
	manifestPath := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	cmd := Command(fs)
	cmd.SetArgs([]string{manifestPath})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load manifest")
}

func TestCommand_FailsIfRequestedProjectIsNotInManifest(t *testing.T) {
	fs, manifestPath := setupTestFs(t)

	cmd := Command(fs)
	cmd.SetArgs([]string{manifestPath, "--project", "unknown-project"})
	err := cmd.Execute()

	assert.ErrorContains(t, err, "failed to load projects")
}

func TestCommand_WritesDeleteFileWithCustomNameToOutputFolder(t *testing.T) {
	fs, manifestPath := setupTestFs(t)
	outputFolder := filepath.Join(t.TempDir(), "out")

	cmd := Command(fs)
	cmd.SetArgs([]string{manifestPath, "--output-folder", outputFolder, "--file", "custom.yaml"})
	require.NoError(t, cmd.Execute())

	entries := readDeleteEntries(t, fs, filepath.Join(outputFolder, "custom.yaml"))
	assert.ElementsMatch(t, []string{"Profile A", "Profile B"}, configNamesOfType(entries, "alerting-profile"))
	assert.ElementsMatch(t, []string{"Dashboard A"}, configNamesOfType(entries, "dashboard"))
}

func TestCommand_ExcludesDashboardShareSettingsByDefault(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
	}{
		{name: "without type filter"},
		// even an explicit include must not produce entries, as share settings cannot be deleted
		{name: "with explicit include", extra: []string{"--types", api.DashboardShareSettings}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, manifestPath := setupTestFs(t)
			outputFolder := filepath.Join(t.TempDir(), "out")

			cmd := Command(fs)
			cmd.SetArgs(append([]string{manifestPath, "--output-folder", outputFolder}, tt.extra...))
			require.NoError(t, cmd.Execute())

			entries := readDeleteEntries(t, fs, filepath.Join(outputFolder, "delete.yaml"))
			assert.Empty(t, configNamesOfType(entries, api.DashboardShareSettings))
		})
	}
}

func TestCommand_RespectsProjectFlag(t *testing.T) {
	fs, manifestPath := setupTestFs(t)
	outputFolder := filepath.Join(t.TempDir(), "out")

	cmd := Command(fs)
	cmd.SetArgs([]string{manifestPath, "--output-folder", outputFolder, "--project", "project-b"})
	require.NoError(t, cmd.Execute())

	entries := readDeleteEntries(t, fs, filepath.Join(outputFolder, "delete.yaml"))
	assert.ElementsMatch(t, []string{"Profile B"}, configNamesOfType(entries, "alerting-profile"))
	assert.Empty(t, configNamesOfType(entries, "dashboard"))
}

func TestCommand_RespectsTypeFilters(t *testing.T) {
	fs, manifestPath := setupTestFs(t)
	outputFolder := filepath.Join(t.TempDir(), "out")

	cmd := Command(fs)
	cmd.SetArgs([]string{manifestPath, "--output-folder", outputFolder, "--types", "alerting-profile,dashboard", "--exclude-types", "dashboard"})
	require.NoError(t, cmd.Execute())

	entries := readDeleteEntries(t, fs, filepath.Join(outputFolder, "delete.yaml"))
	assert.ElementsMatch(t, []string{"Profile A", "Profile B"}, configNamesOfType(entries, "alerting-profile"))
	assert.Empty(t, configNamesOfType(entries, "dashboard"))
}
