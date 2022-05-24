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
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

func createTestConfig(name string, filePrefix string, property string) config.Config {

	propA := make(map[string]map[string]string)
	propA[name] = make(map[string]string)
	propA[name]["firstProp"] = "foo"
	propA[name]["secondProp"] = property

	path := strings.Split(filePrefix, string(os.PathSeparator))
	zoneId := path[len(path)-2 : len(path)-1]
	project := strings.Join(path[0:len(path)-2], string(os.PathSeparator))
	fileReaderMock := util.CreateTestFileSystem()
	var testManagementZoneApi = api.NewStandardApi(zoneId[0], "/api/config/v1/foobar", false)

	configA := config.GetMockConfig(fileReaderMock, name, project, nil, propA, testManagementZoneApi, filePrefix+name+".json")

	return configA
}

func TestSortingByConfigDependencyWithRootDirectory(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, "foo")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")

	configs := []config.Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)

	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestFailsOnCircularConfigDependency(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathB+"profile.name")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")

	configs := []config.Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.Error(t, err, "failed to sort configs, circular dependency on config "+pathB+"profile detected, please check dependencies")

	assert.Check(t, configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestSortingByConfigDependencyWithoutRootDirectory(t *testing.T) {

	pathA := util.ReplacePathSeparators("infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("infrastructure/synthetic/")
	configA := createTestConfig("zone-d", pathA, "bar")
	configB := createTestConfig("profile", pathB, pathA+"zone-d.id")

	configs := []config.Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)
	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestSortingByConfigDependencyWithRelativePath(t *testing.T) {

	pathA := util.ReplacePathSeparators("infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("infrastructure/synthetic/")
	configA := createTestConfig("testzone", pathA, "prop")
	configB := createTestConfig("profile", pathB, "management-zone"+string(os.PathSeparator)+"testzone.id")

	configs := []config.Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)
	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestFailsOnCircularProjectDependency(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	pathC := util.ReplacePathSeparators("my-project/management-zone/")
	configA := createTestConfig("zone-a", pathA, "foo")
	configB := createTestConfig("profile", pathB, pathC+"zone-b.id")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA},
	}

	pathD := util.ReplacePathSeparators("my-project/alerting-profile/")
	configC := createTestConfig("zone-b", pathC, "foo")
	configD := createTestConfig("profile", pathD, pathA+"zone-a.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configC, configD},
	}

	projects := []Project{projectB, projectA} // reverse ordering

	// Our assumption of dependencies
	assert.Check(t, projectA.HasDependencyOn(projectB))
	assert.Check(t, projectB.HasDependencyOn(projectA))

	// sort.Sort(byProjectDependency(projects))
	projects, err := sortProjects(projects)

	assert.Error(t, err, "failed to sort projects, circular dependency on project B detected, please check dependencies in project configs")
}

func TestSortingByProjectDependency_1(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, "foo")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA},
	}

	pathC := util.ReplacePathSeparators("my-project/management-zone/")
	pathD := util.ReplacePathSeparators("my-project/alerting-profile/")
	configC := createTestConfig("zone-a", pathC, "foo")
	configD := createTestConfig("profile", pathD, pathA+"zone-a.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configC, configD},
	}

	projects := []Project{projectB, projectA} // reverse ordering

	// Our assumption of dependencies
	assert.Check(t, !projectA.HasDependencyOn(projectB))
	assert.Check(t, projectB.HasDependencyOn(projectA))

	// sort.Sort(byProjectDependency(projects))
	projects, err := sortProjects(projects)
	assert.NilError(t, err)
	// After sort we expect {A, B}
	assert.Equal(t, projectA, projects[0])
	assert.Equal(t, projectB, projects[1])
}

func TestSortingByProjectDependency_2(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	pathX := util.ReplacePathSeparators("projects/token/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathX+"later.id")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA},
	}

	pathD := util.ReplacePathSeparators("projects/my-project/alerting-profile/")
	configD := createTestConfig("profile", pathD, pathA+"zone-a.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configD},
	}

	configX := createTestConfig("later", pathX, "special")
	projectX := &projectImpl{
		id:      "X",
		configs: []config.Config{configX},
	}

	projects := []Project{projectB, projectA, projectX}

	// Our assumption of dependencies A needs X, B needs A
	assert.Check(t, !projectA.HasDependencyOn(projectB))
	assert.Check(t, projectA.HasDependencyOn(projectX))
	assert.Check(t, projectB.HasDependencyOn(projectA))
	assert.Check(t, !projectB.HasDependencyOn(projectX))
	assert.Check(t, !projectX.HasDependencyOn(projectA))
	assert.Check(t, !projectX.HasDependencyOn(projectB))

	// sort.Sort(byProjectDependency(projects))
	projects, err := sortProjects(projects)
	assert.NilError(t, err)
	// After sort we expect {X, A, B}
	assert.Equal(t, projectX, projects[0])
	assert.Equal(t, projectA, projects[1])
	assert.Equal(t, projectB, projects[2])
}

func TestSortingByProjectDependency_3(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	pathX := util.ReplacePathSeparators("projects/token/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathX+"profileX.id")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA},
	}

	pathD := util.ReplacePathSeparators("projects/my-project/alerting-profile/")
	configD := createTestConfig("profile", pathD, pathA+"zone-a.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configD},
	}

	configX := createTestConfig("profileX", pathX, "special")
	configX2 := createTestConfig("depOnY", pathX, pathX+"profileY.name")
	projectX := &projectImpl{
		id:      "X",
		configs: []config.Config{configX, configX2},
	}

	configY := createTestConfig("profileY", pathX, "special")
	projectY := &projectImpl{
		id:      "Y",
		configs: []config.Config{configY},
	}

	projects := []Project{projectB, projectY, projectA, projectX}

	assert.Check(t, !projectA.HasDependencyOn(projectB))
	assert.Check(t, projectA.HasDependencyOn(projectX))
	assert.Check(t, !projectA.HasDependencyOn(projectY))

	assert.Check(t, projectB.HasDependencyOn(projectA))
	assert.Check(t, !projectB.HasDependencyOn(projectX))
	assert.Check(t, !projectB.HasDependencyOn(projectY))

	assert.Check(t, !projectX.HasDependencyOn(projectA))
	assert.Check(t, !projectX.HasDependencyOn(projectB))
	assert.Check(t, projectX.HasDependencyOn(projectY))

	assert.Check(t, !projectY.HasDependencyOn(projectA))
	assert.Check(t, !projectY.HasDependencyOn(projectB))
	assert.Check(t, !projectY.HasDependencyOn(projectX))

	// sort.Stable(byProjectDependency(projects))
	projects, err := sortProjects(projects)

	for i := 0; i < len(projects); i++ {
		println(projects[i].GetId())
	}
	assert.NilError(t, err)
	assert.Equal(t, projectY, projects[0])
	assert.Equal(t, projectX, projects[1])
	assert.Equal(t, projectA, projects[2])
	assert.Equal(t, projectB, projects[3])
}

func TestSortingByProjectDependency_4(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	pathX := util.ReplacePathSeparators("projects/token/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathX+"later.id")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA},
	}

	configD := createTestConfig("profile", pathA, pathX+"later.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configD},
	}

	configX := createTestConfig("later", pathX, "special")
	projectX := &projectImpl{
		id:      "X",
		configs: []config.Config{configX},
	}

	projects := []Project{projectB, projectA, projectX}

	// Our assumption of dependencies A needs X, B needs X
	assert.Check(t, !projectA.HasDependencyOn(projectB))
	assert.Check(t, projectA.HasDependencyOn(projectX))
	assert.Check(t, !projectB.HasDependencyOn(projectA))
	assert.Check(t, projectB.HasDependencyOn(projectX))
	assert.Check(t, !projectX.HasDependencyOn(projectA))
	assert.Check(t, !projectX.HasDependencyOn(projectB))

	// sort.Sort(byProjectDependency(projects))
	projects, err := sortProjects(projects)
	assert.NilError(t, err)
	// After sort we expect {X, A, B}
	assert.Equal(t, projectX, projects[0])
	assert.Equal(t, projectA, projects[1])
	assert.Equal(t, projectB, projects[2])
}

func TestSortingByProjectDependency_5(t *testing.T) {

	pathA := util.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := util.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	pathX := util.ReplacePathSeparators("projects/token/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathX+"profileX.id")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")
	configC := createTestConfig("depOnY", pathX, pathX+"profileY.name")
	projectA := &projectImpl{
		id:      "A",
		configs: []config.Config{configB, configA, configC},
	}

	pathD := util.ReplacePathSeparators("projects/my-project/alerting-profile/")
	configD := createTestConfig("profile", pathD, pathA+"zone-a.id")
	projectB := &projectImpl{
		id:      "B",
		configs: []config.Config{configD},
	}

	configX := createTestConfig("profileX", pathX, "special")
	projectX := &projectImpl{
		id:      "X",
		configs: []config.Config{configX},
	}

	configY := createTestConfig("profileY", pathX, "special")
	projectY := &projectImpl{
		id:      "Y",
		configs: []config.Config{configY},
	}

	projects := []Project{projectB, projectY, projectA, projectX}

	// Assert dependency assumptions
	assert.Check(t, !projectA.HasDependencyOn(projectB))
	assert.Check(t, projectA.HasDependencyOn(projectX))
	assert.Check(t, projectA.HasDependencyOn(projectY))

	assert.Check(t, projectB.HasDependencyOn(projectA))
	assert.Check(t, !projectB.HasDependencyOn(projectX))
	assert.Check(t, !projectB.HasDependencyOn(projectY))

	assert.Check(t, !projectX.HasDependencyOn(projectA))
	assert.Check(t, !projectX.HasDependencyOn(projectB))
	assert.Check(t, !projectX.HasDependencyOn(projectY))

	assert.Check(t, !projectY.HasDependencyOn(projectA))
	assert.Check(t, !projectY.HasDependencyOn(projectB))
	assert.Check(t, !projectY.HasDependencyOn(projectX))

	projects, err := sortProjects(projects)

	for i := 0; i < len(projects); i++ {
		println(projects[i].GetId())
	}
	assert.NilError(t, err)
	assert.Equal(t, projectX, projects[0])
	assert.Equal(t, projectY, projects[1])
	assert.Equal(t, projectA, projects[2])
	assert.Equal(t, projectB, projects[3])
}
