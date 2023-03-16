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

package convert

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/version"
	"github.com/spf13/afero"
	"gotest.tools/assert"
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestConvert_WorksOnFullConfiguration(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("config:\n  - profile: \"profile.json\"\n\nprofile:\n  - name: \"Star Trek Service\""), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "environments.yaml", []byte("env:\n  - name: \"My_Environment\"\n  - env-url: \"{{ .Env.ENV_URL }}\"\n  - env-token-name: \"ENV_TOKEN\""), 0644)
	_ = afero.WriteFile(testFs, "delete.yaml", []byte("delete:\n-\"some/config\""), 0644)

	err := convert(testFs, ".", "environments.yaml", "converted", "manifest.yaml")
	assert.NilError(t, err)

	outputFolderExists, _ := afero.Exists(testFs, "converted/")
	assert.Check(t, outputFolderExists)

	assertExpectedConfigurationCreated(t, testFs)

	assertExpectedManifestCreated(t, testFs)

	assertExpectedDeleteFileCreated(t, testFs)
}

func TestConvert_WorksIfNoDeleteYamlExists(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.yaml", []byte("config:\n  - profile: \"profile.json\"\n\nprofile:\n  - name: \"Star Trek Service\""), 0644)
	_ = afero.WriteFile(testFs, "project/alerting-profile/profile.json", []byte("{}"), 0644)
	_ = afero.WriteFile(testFs, "environments.yaml", []byte("env:\n  - name: \"My_Environment\"\n  - env-url: \"{{ .Env.ENV_URL }}\"\n  - env-token-name: \"ENV_TOKEN\""), 0644)

	err := convert(testFs, ".", "environments.yaml", "converted", "manifest.yaml")
	assert.NilError(t, err)

	outputFolderExists, _ := afero.Exists(testFs, "converted/")
	assert.Check(t, outputFolderExists)

	assertExpectedConfigurationCreated(t, testFs)

	assertExpectedManifestCreated(t, testFs)
}

func TestConvert_FailsIfThereIsJustEmptyProjects(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("project/", 0755)
	_ = afero.WriteFile(testFs, "environments.yaml", []byte("env:\n  - name: \"My_Environment\"\n  - env-url: \"{{ .Env.ENV_URL }}\"\n  - env-token-name: \"ENV_TOKEN\""), 0644)

	err := convert(testFs, ".", "environments.yaml", "converted", "manifest.yaml")
	assert.ErrorContains(t, err, "no projects to convert")
}

func assertExpectedConfigurationCreated(t *testing.T, testFs afero.Fs) {
	outputConfigExists, _ := afero.Exists(testFs, "converted/project/alerting-profile/config.yaml")
	assert.Check(t, outputConfigExists)
	configContent, err := afero.ReadFile(testFs, "converted/project/alerting-profile/config.yaml")
	assert.NilError(t, err)
	assert.Equal(t, string(configContent), "configs:\n- id: profile\n  config:\n    name: Star Trek Service\n    template: profile.json\n    skip: false\n  type:\n    api: alerting-profile\n")

	outputPayloadExists, _ := afero.Exists(testFs, "converted/project/alerting-profile/profile.json")
	assert.Check(t, outputPayloadExists)
	payloadContent, err := afero.ReadFile(testFs, "converted/project/alerting-profile/profile.json")
	assert.NilError(t, err)
	assert.Equal(t, string(payloadContent), "{}")
}

func assertExpectedManifestCreated(t *testing.T, testFs afero.Fs) {
	expectedManifest := fmt.Sprintf(
		`manifestVersion: "%s"
projects:
- name: project
environmentGroups:
- name: default
  environments:
  - name: env
    url:
      type: environment
      value: ENV_URL
    auth:
      token:
        type: environment
        name: ENV_TOKEN
`, version.ManifestVersion)

	manifestExists, _ := afero.Exists(testFs, "converted/manifest.yaml")
	assert.Check(t, manifestExists)
	manifestContent, err := afero.ReadFile(testFs, "converted/manifest.yaml")
	assert.NilError(t, err)
	assert.Equal(t, string(manifestContent), expectedManifest)
}

func assertExpectedDeleteFileCreated(t *testing.T, testFs afero.Fs) {
	deleteExists, _ := afero.Exists(testFs, "converted/delete.yaml")
	assert.Check(t, deleteExists)
	deleteContent, err := afero.ReadFile(testFs, "converted/delete.yaml")
	assert.NilError(t, err)
	assert.Equal(t, string(deleteContent), "delete:\n-\"some/config\"")
}

func TestCopyDeleteFileIfPresent_copiesDeleteFile(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = afero.WriteFile(testFs, "project/delete.yaml", []byte("delete:\n-\"some/config\""), 0644)

	err := copyDeleteFileIfPresent(testFs, "project", "new_project")
	assert.NilError(t, err)

	deleteFileExistsInOutputFolder, err := afero.Exists(testFs, "new_project/delete.yaml")
	assert.Check(t, deleteFileExistsInOutputFolder)
	assert.NilError(t, err)
}

func TestCopyDeleteFileIfPresent_doesNothingIfNoFileIsFound(t *testing.T) {
	testFs := afero.NewMemMapFs()
	_ = testFs.MkdirAll("project/", 0755)

	err := copyDeleteFileIfPresent(testFs, "project", "new_project")
	assert.NilError(t, err)

	deleteFileExistsInOutputFolder, err := afero.Exists(testFs, "new_project/delete.yaml")
	assert.Check(t, !deleteFileExistsInOutputFolder)
	assert.NilError(t, err)
}

func TestCopyDeleteFileIfPresent_returnsErrorIfFileCanNotBeAccessed(t *testing.T) {
	testFs := inaccessibleMockFs{}
	testFs.inaccessiblePath = "project/delete.yaml"

	err := copyDeleteFileIfPresent(&testFs, "project", "new_project")
	assert.ErrorContains(t, err, "permission denied")
}

func TestCopyDeleteFileIfPresent_returnsErrorIfFileCanNotBeRead(t *testing.T) {
	testFs := inaccessibleMockFs{}
	_ = afero.WriteFile(&testFs, "project/delete.yaml", []byte("delete:\n-\"some/config\""), 0644)
	testFs.inaccessiblePath = "project/delete.yaml"
	testFs.filePathExistsButCantBeOpened = true

	err := copyDeleteFileIfPresent(&testFs, "project", "new_project")
	assert.ErrorContains(t, err, "permission denied")
}

func TestCopyDeleteFileIfPresent_returnsErrorIfOutputFolderCanNotBeAccessed(t *testing.T) {
	testFs := inaccessibleMockFs{}
	_ = afero.WriteFile(&testFs, "project/delete.yaml", []byte("delete:\n-\"some/config\""), 0644)
	testFs.inaccessiblePath = "new_project/"

	err := copyDeleteFileIfPresent(&testFs, "project", "new_project")
	assert.ErrorContains(t, err, "permission denied")
}

// This is needed to test failed/denied access error cases, as afero.MemMapFs does not implement permissions
// See also https://github.com/spf13/afero/issues/150
type inaccessibleMockFs struct {
	inaccessiblePath              string
	filePathExistsButCantBeOpened bool
	afero.MemMapFs
}

func (f *inaccessibleMockFs) Open(name string) (afero.File, error) {
	if f.isOnInaccessiblePath(name) {
		return nil, fs.ErrPermission
	}
	return f.MemMapFs.Open(name)
}

func (f *inaccessibleMockFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if f.isOnInaccessiblePath(name) {
		return nil, fs.ErrPermission
	}
	return f.MemMapFs.OpenFile(name, flag, perm)
}

func (f *inaccessibleMockFs) Stat(name string) (fs.FileInfo, error) {
	if !f.filePathExistsButCantBeOpened && f.isOnInaccessiblePath(name) {
		return nil, fs.ErrPermission
	}
	return f.MemMapFs.Stat(name)
}

func (f *inaccessibleMockFs) isOnInaccessiblePath(file string) bool {
	return len(f.inaccessiblePath) > 0 && strings.HasPrefix(file, f.inaccessiblePath)
}
