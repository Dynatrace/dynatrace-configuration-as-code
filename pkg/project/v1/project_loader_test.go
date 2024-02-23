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

package v1

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
)

func TestIfProjectHasSubproject(t *testing.T) {
	mt := files.ReplacePathSeparators("marvin/trillian")
	mth := files.ReplacePathSeparators("marvin/trillian/hacktar")
	rth := files.ReplacePathSeparators("robot/trillian/hacktar")
	projects := []string{"zem", "marvin", mt, mth, rth}
	assert.True(t, hasSubprojectFolder("marvin", projects), "Check if `marvin` project has subprojects")
	assert.True(t, hasSubprojectFolder(mt, projects), "Check if `marvin/trillian` project has subprojects")
	assert.False(t, hasSubprojectFolder(mth, projects), "Check if `marvin/trillian` project has subprojects")
	assert.False(t, hasSubprojectFolder(rth, projects), "Check if `marvin/trillian` project has subprojects")
	assert.False(t, hasSubprojectFolder("zem", projects), "Check if `zem` project has subprojects")
	assert.False(t, hasSubprojectFolder("unknown", projects), "Check if `zem` project has subprojects")
}

func TestFilterProjectsWithSubproject(t *testing.T) {
	ca := files.ReplacePathSeparators("caveman/anjie")
	cag := files.ReplacePathSeparators("caveman/anjie/garkbit")
	mt := files.ReplacePathSeparators("marvin/trillian")
	allProjectFolders := []string{"zem", ca, cag, mt, "trillian"}
	allProjectFolders = filterProjectsWithSubproject(allProjectFolders)

	assert.Equal(t, "zem", allProjectFolders[0], "Check if `zem` folder in list")
	assert.Equal(t, cag, allProjectFolders[1], "Check if `caveman/anjie/garkbit` folder in list")
	assert.Equal(t, mt, allProjectFolders[2], "Check if `marvin/trillian` folder in list")
	assert.Equal(t, "trillian", allProjectFolders[3], "Check if `trillian` folder in list")
	assert.Len(t, allProjectFolders, 4, "Check if only 4 project folders are returned.")
}

func TestGetAllProjectFoldersRecursivelyFailsOnMixedFolder(t *testing.T) {
	path := files.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project1")
	fs := testutils.CreateTestFileSystem()
	apis := api.NewAPIs()
	_, err := getAllProjectFoldersRecursively(fs, apis, path)

	expected := files.ReplacePathSeparators("found folder with projects and configurations in test-resources/configs-and-api-mixed-test/project1")
	require.Error(t, err, expected)
}

func TestGetAllProjectFoldersRecursivelyFailsOnMixedFolderInSubproject(t *testing.T) {
	path := files.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project2")
	fs := testutils.CreateTestFileSystem()
	apis := api.NewAPIs()
	_, err := getAllProjectFoldersRecursively(fs, apis, path)

	expected := files.ReplacePathSeparators("found folder with projects and configurations in test-resources/configs-and-api-mixed-test/project2/subproject2")
	require.Error(t, err, expected)
}

func TestGetAllProjectFoldersRecursivelyPassesOnSeparatedFolders(t *testing.T) {
	path := files.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project3")
	fs := testutils.CreateTestFileSystem()
	apis := api.NewAPIs()
	_, err := getAllProjectFoldersRecursively(fs, apis, path)
	require.NoError(t, err)
}

func TestGetAllProjectsFoldersRecursivelyPassesOnHiddenFolders(t *testing.T) {
	path := files.ReplacePathSeparators("test-resources/hidden-directories/project1")
	fs := testutils.CreateTestFileSystem()
	_, err := getAllProjectFoldersRecursively(fs, api.NewV1APIs(), path)
	require.NoError(t, err)
}

func TestGetAllProjectsFoldersRecursivelyPassesOnProjectsWithinHiddenFolders(t *testing.T) {
	path := filepath.FromSlash("test-resources/hidden-directories/project2")
	fs := testutils.CreateTestFileSystem()
	projects, err := getAllProjectFoldersRecursively(fs, api.NewV1APIs(), path)

	require.NoError(t, err)

	// NOT test-resources/hidden-directories/project2/.logs
	assert.Equal(t, []string{filepath.FromSlash("test-resources/hidden-directories/project2/subproject")}, projects)
}

func TestGetAllProjectsFoldersRecursivelyPassesOnProjects(t *testing.T) {
	path := filepath.FromSlash("test-resources/hidden-directories")
	fs := testutils.CreateTestFileSystem()
	projects, err := getAllProjectFoldersRecursively(fs, api.NewV1APIs(), path)

	require.NoError(t, err)

	// NOT test-resources/hidden-directories/.logs
	// NOT test-resources/hidden-directories/project2/.logs

	assert.ElementsMatch(t, projects, []string{
		filepath.FromSlash("test-resources/hidden-directories/project1"),
		filepath.FromSlash("test-resources/hidden-directories/project2/subproject"),
	})
}

func TestContainsApiName(t *testing.T) {
	apis := api.NewAPIs()
	assert.False(t, containsApiName(apis, "trillian"), "Check if `trillian` is an API")
	assert.True(t, containsApiName(apis, "extension"), "Check if `extension` is an API")
	assert.True(t, containsApiName(apis, "/project/sub-project/extension/subfolder"), "Check if `extension` is an API")
	assert.False(t, containsApiName(apis, "/project/sub-project"), "Check if `extension` is an API")
}
