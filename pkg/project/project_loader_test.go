//go:build unit
// +build unit

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

package project

import (
	"os"
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func TestIfProjectHasSubproject(t *testing.T) {
	mt := util.ReplacePathSeparators("marvin/trillian")
	mth := util.ReplacePathSeparators("marvin/trillian/hacktar")
	rth := util.ReplacePathSeparators("robot/trillian/hacktar")
	projects := []string{"zem", "marvin", mt, mth, rth}
	assert.Equal(t, hasSubprojectFolder("marvin", projects), true, "Check if `marvin` project has subprojects")
	assert.Equal(t, hasSubprojectFolder(mt, projects), true, "Check if `marvin/trillian` project has subprojects")
	assert.Equal(t, hasSubprojectFolder(mth, projects), false, "Check if `marvin/trillian` project has subprojects")
	assert.Equal(t, hasSubprojectFolder(rth, projects), false, "Check if `marvin/trillian` project has subprojects")
	assert.Equal(t, hasSubprojectFolder("zem", projects), false, "Check if `zem` project has subprojects")
	assert.Equal(t, hasSubprojectFolder("unknown", projects), false, "Check if `zem` project has subprojects")
}

func TestCreateProjectsFromFolderList(t *testing.T) {
	path := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	specificProjectToDeploy := "zem, marvin, caveman"
	apis := api.NewApis()
	fs := util.CreateTestFileSystem()
	allProjectFolders, err := getAllProjectFoldersRecursively(fs, path)
	assert.NilError(t, err)

	projects, err := createProjectsListFromFolderList(fs, path, specificProjectToDeploy, path, apis, allProjectFolders)

	assert.NilError(t, err)

	ps := string(os.PathSeparator)
	assert.Equal(t, projects[0].GetId(), path+ps+"zem", "Check if `zem` in projects list")
	assert.Equal(t, projects[1].GetId(), path+ps+"marvin", "Check if `marvin` in projects list")
	assert.Equal(t, projects[2].GetId(), path+ps+"caveman"+ps+"anjie"+ps+"garkbit", "Check if `caveman/anjie/garkbit` in projects list")
	assert.Equal(t, projects[3].GetId(), path+ps+"caveman"+ps+"eddie", "Check if `eddie` in projects list")
	assert.Equal(t, len(projects), 4, "Check if there are only 4 projects in the list.")
}

func TestLoadProjectsToDeployFromFolder(t *testing.T) {
	folder := "test-resources/transitional-dependency-test"
	fs := util.CreateTestFileSystem()
	projects, err := LoadProjectsToDeploy(fs, "", api.NewApis(), folder)
	assert.NilError(t, err)
	assert.Equal(t, len(projects), 7, "Check if all projects are loaded into list.")
}

func TestLoadProjectsThrowsErrorOnCircularConfigDependecy(t *testing.T) {
	folder := "test-resources/circular-config-dependency-test"
	fs := util.CreateTestFileSystem()
	_, err := LoadProjectsToDeploy(fs, "", api.NewApis(), folder)
	assert.ErrorContains(t, err, "circular dependency on config")
}

func TestLoadProjectsThrowsErrorOnCircularProjectDependency(t *testing.T) {
	folder := "test-resources/circular-project-dependency-test"
	fs := util.CreateTestFileSystem()
	_, err := LoadProjectsToDeploy(fs, "", api.NewApis(), folder)
	assert.ErrorContains(t, err, "circular dependency on project")
}

/*Test loading of project aseed
 * Dependencies: aseed -> marvin & trillian, marvin -> tillian, trillian -> zaphod
 * Expected sorted projects: zaphod, trillian, marvin, asseed
 */
func TestLoadProjectsToDeployWithTransitionalDependencies(t *testing.T) {
	folder := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	fs := util.CreateTestFileSystem()
	projects, err := LoadProjectsToDeploy(fs, "aseed", api.NewApis(), folder)

	assert.NilError(t, err)

	assert.Equal(t, len(projects), 4, "Check if there are only 4 projects in the list.")

	ps := string(os.PathSeparator)
	assert.Equal(t, projects[0].GetId(), folder+ps+"zaphod", "Check if `zaphod` in projects list")
	assert.Equal(t, projects[1].GetId(), folder+ps+"trillian", "Check if `trillian` in projects list")
	assert.Equal(t, projects[2].GetId(), folder+ps+"marvin", "Check if `marvin` in projects list")
	assert.Equal(t, projects[3].GetId(), folder+ps+"aseed", "Check if `aseed` in projects list")
}

/*Test loading of project zem
 * Dependencies: zem -> caveman/eddie, caveman/eddie -> zaphod
 * Expected sorted projects: zaphod, caveman/eddie, zem
 */
func TestLoadProjectsWithResolvingDependenciesInProjectsTree1(t *testing.T) {
	folder := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	fs := util.CreateTestFileSystem()
	projects, err := LoadProjectsToDeploy(fs, "zem", api.NewApis(), folder)

	assert.NilError(t, err)

	assert.Equal(t, len(projects), 3, "Check if there are only 3 projects in the list.")

	ps := string(os.PathSeparator)
	assert.Equal(t, projects[0].GetId(), folder+ps+"zaphod", "Check if `zaphod` in projects list")
	assert.Equal(t, projects[1].GetId(), folder+ps+"caveman"+ps+"eddie", "Check if `caveman/eddie` in projects list")
	assert.Equal(t, projects[2].GetId(), folder+ps+"zem", "Check if `zem` in projects list")
}

/*Test loading of projects zem, marvin, caveman
 * Dependencies: zem -> caveman/eddie, caveman/eddie -> zaphod, marvin -> tillian, trillian -> zaphod, caveman/anjie/garkbit -> trillian
 * Expected sorted projects: zaphod, trillian, caveman, zem
 */
func TestLoadProjectsWithResolvingDependenciesInProjectsTree2(t *testing.T) {
	folder := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	fs := util.CreateTestFileSystem()
	projects, err := LoadProjectsToDeploy(fs, "zem, marvin, caveman", api.NewApis(), folder)

	assert.NilError(t, err)

	assert.Equal(t, len(projects), 6, "Check if there are 6 projects in the list.")

	ps := string(os.PathSeparator)
	assert.Equal(t, projects[5].GetId(), folder+ps+"zem", "Check if `zem` in projects list")
	assert.Equal(t, projects[4].GetId(), folder+ps+"marvin", "Check if `marvin` in projects list")
	assert.Equal(t, projects[3].GetId(), folder+ps+"caveman"+ps+"anjie"+ps+"garkbit", "Check if `caveman/anjie/garkbit` in projects list")
	assert.Equal(t, projects[2].GetId(), folder+ps+"caveman"+ps+"eddie", "Check if `caveman/eddie` in projects list")
	assert.Equal(t, projects[1].GetId(), folder+ps+"trillian", "Check if `trillian` in projects list")
	assert.Equal(t, projects[0].GetId(), folder+ps+"zaphod", "Check if `zaphod` in projects list")
}

func TestLoadProjectsWithResolvingDependenciesInProjectsTreeProjectSubprojectWithoutDependencies(t *testing.T) {
	folder := util.ReplacePathSeparators("test-resources/transitional-dependency-test")
	project := util.ReplacePathSeparators("caveman/anjie/garkbit")
	fs := util.CreateTestFileSystem()
	projects, err := LoadProjectsToDeploy(fs, project, api.NewApis(), folder)

	assert.NilError(t, err)

	assert.Equal(t, projects[0].GetId(), folder+string(os.PathSeparator)+project, "Check if `caveman/anjie/garkbit` in projects list")
	assert.Equal(t, len(projects), 1, "Check if there is only 1 project in the list.")
}

func TestFilterProjectsWithSubproject(t *testing.T) {
	ca := util.ReplacePathSeparators("caveman/anjie")
	cag := util.ReplacePathSeparators("caveman/anjie/garkbit")
	mt := util.ReplacePathSeparators("marvin/trillian")
	allProjectFolders := []string{"zem", ca, cag, mt, "trillian"}
	allProjectFolders = filterProjectsWithSubproject(allProjectFolders)

	assert.Equal(t, allProjectFolders[0], "zem", "Check if `zem` folder in list")
	assert.Equal(t, allProjectFolders[1], cag, "Check if `caveman/anjie/garkbit` folder in list")
	assert.Equal(t, allProjectFolders[2], mt, "Check if `marvin/trillian` folder in list")
	assert.Equal(t, allProjectFolders[3], "trillian", "Check if `trillian` folder in list")
	assert.Equal(t, len(allProjectFolders), 4, "Check if only 4 project folders are returned.")
}

func TestGetAllProjectFoldersRecursivelyFailsOnMixedFolder(t *testing.T) {
	path := util.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project1")
	fs := util.CreateTestFileSystem()
	_, err := getAllProjectFoldersRecursively(fs, path)

	expected := util.ReplacePathSeparators("found folder with projects and configurations in test-resources/configs-and-api-mixed-test/project1")
	assert.Error(t, err, expected)
}

func TestGetAllProjectFoldersRecursivelyFailsOnMixedFolderInSubproject(t *testing.T) {
	path := util.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project2")
	fs := util.CreateTestFileSystem()
	_, err := getAllProjectFoldersRecursively(fs, path)

	expected := util.ReplacePathSeparators("found folder with projects and configurations in test-resources/configs-and-api-mixed-test/project2/subproject2")
	assert.Error(t, err, expected)
}

func TestGetAllProjectFoldersRecursivelyPassesOnSeparatedFolders(t *testing.T) {
	path := util.ReplacePathSeparators("test-resources/configs-and-api-mixed-test/project3")
	fs := util.CreateTestFileSystem()
	_, err := getAllProjectFoldersRecursively(fs, path)
	assert.NilError(t, err)
}
