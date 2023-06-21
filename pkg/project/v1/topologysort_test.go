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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"gotest.tools/assert"
	"os"
	"strings"
	"testing"
)

func createTestConfig(name string, filePrefix string, property string) *Config {

	propA := make(map[string]map[string]string)
	propA[name] = make(map[string]string)
	propA[name]["firstProp"] = "foo"
	propA[name]["secondProp"] = property

	path := strings.Split(filePrefix, string(os.PathSeparator))
	zoneId := path[len(path)-2 : len(path)-1]
	project := strings.Join(path[0:len(path)-2], string(os.PathSeparator))
	var testManagementZoneApi = api.API{ID: zoneId[0], URLPath: "/api/config/v1/foobar"}

	configA := NewConfigWithTemplate(name, project, filePrefix+name+".json", nil, propA, testManagementZoneApi)

	return configA
}

func TestSortingByConfigDependencyWithRootDirectory(t *testing.T) {

	pathA := files.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := files.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, "foo")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")

	configs := []*Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)

	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestFailsOnCircularConfigDependency(t *testing.T) {

	pathA := files.ReplacePathSeparators("projects/infrastructure/management-zone/")
	pathB := files.ReplacePathSeparators("projects/infrastructure/alerting-profile/")
	configA := createTestConfig("zone-a", pathA, pathB+"profile.name")
	configB := createTestConfig("profile", pathB, pathA+"zone-a.id")

	configs := []*Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.Error(t, err, "failed to sort configs, circular dependency on config "+pathB+"profile detected, please check dependencies")

	assert.Check(t, configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestSortingByConfigDependencyWithoutRootDirectory(t *testing.T) {

	pathA := files.ReplacePathSeparators("infrastructure/management-zone/")
	pathB := files.ReplacePathSeparators("infrastructure/synthetic/")
	configA := createTestConfig("zone-d", pathA, "bar")
	configB := createTestConfig("profile", pathB, pathA+"zone-d.id")

	configs := []*Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)
	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}

func TestSortingByConfigDependencyWithRelativePath(t *testing.T) {

	pathA := files.ReplacePathSeparators("infrastructure/management-zone/")
	pathB := files.ReplacePathSeparators("infrastructure/synthetic/")
	configA := createTestConfig("testzone", pathA, "prop")
	configB := createTestConfig("profile", pathB, "management-zone"+string(os.PathSeparator)+"testzone.id")

	configs := []*Config{configB, configA} // reverse ordering

	configs, err := sortConfigurations(configs)
	assert.NilError(t, err)
	assert.Equal(t, configA, configs[0])
	assert.Equal(t, configB, configs[1])

	assert.Check(t, !configA.HasDependencyOn(configB))
	assert.Check(t, configB.HasDependencyOn(configA))
}
