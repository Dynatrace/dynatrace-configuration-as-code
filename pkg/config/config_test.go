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

package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

const testTemplate = `{"msg": "Follow the {{.color}} {{.animalType}}"}`

var testProductionEnvironment = environment.NewEnvironment("prod-environment", "prod-environment", "production", "https://url/to/production/environment", "PRODUCTION")
var testManagementZoneApi = api.NewStandardApi("management-zone", "/api/config/v1/managementZones", false, "", false)

func createConfigForTest(id string, project string, template util.Template, properties map[string]map[string]string, api api.Api, fileName string) configImpl {
	return configImpl{
		id:         id,
		project:    project,
		template:   template,
		properties: properties,
		api:        api,
		fileName:   fileName,
	}
}

func TestFilterProperties(t *testing.T) {

	m := make(map[string]map[string]string)

	m["Captains"] = make(map[string]string)
	m["Commanders"] = make(map[string]string)

	m["Captains"]["Kirk"] = "James T."
	m["Captains"]["Picard"] = "Jean Luc"

	m["Commanders"]["Bonaparte"] = "Napoleon"

	properties := filterProperties("Captains", m)

	assert.Check(t, len(properties) == 1)
	assert.Check(t, properties["Captains"] != nil)
}

func TestFilterPropertiesToReturnExactMatchOnlyForConfigName(t *testing.T) {
	m := make(map[string]map[string]string)

	m["dashboard"] = make(map[string]string)
	m["dashboard-availability"] = make(map[string]string)

	properties := filterProperties("dashboard", m)

	assert.Check(t, len(properties) == 1)
	assert.Check(t, properties["dashboard"] != nil)
}

func TestFilterPropertiesToReturnExactMatchOnlyForConfigNameAndEnvironment(t *testing.T) {
	m := make(map[string]map[string]string)

	m["dashboard"] = make(map[string]string)
	m["dashboard-availability"] = make(map[string]string)
	m["dashboard.dev"] = make(map[string]string)
	m["dashboard-availability.dev"] = make(map[string]string)

	m["dashboard"]["prop1"] = "A"
	m["dashboard"]["prop2"] = "A"
	m["dashboard-availability"]["prop1"] = "A"
	m["dashboard-availability"]["prop2"] = "A"

	m["dashboard.dev"]["prop1"] = "B"
	m["dashboard.dev"]["prop2"] = "C"
	m["dashboard-availability.dev"]["prop1"] = "E"
	m["dashboard-availability.dev"]["prop2"] = "F"

	properties := filterProperties("dashboard.dev", m)

	assert.Check(t, len(properties) == 1)
	assert.Check(t, properties["dashboard.dev"] != nil)
	assert.Check(t, len(properties["dashboard.dev"]) == 2)
}

func TestFilterPropertiesToReturnMoreSpecificProperties(t *testing.T) {
	m := make(map[string]map[string]string)

	m["dashboard"] = make(map[string]string)
	m["dashboard.dev"] = make(map[string]string)

	// General properties for all environments
	m["dashboard"]["prop1"] = "A"
	m["dashboard"]["prop2"] = "A"

	// Specific properties for dev environment. Note that "prop2" is missing here.
	m["dashboard.dev"]["prop1"] = "B"
	m["dashboard.dev"]["prop2"] = "C"

	properties := filterProperties("dashboard.dev", m)

	assert.Check(t, properties["dashboard.dev"]["prop1"] == "B")
	assert.Check(t, properties["dashboard.dev"]["prop2"] == "C")
}

func TestFilterPropertiesToReturnNoGeneralPropertiesForMissingSpecificOnes(t *testing.T) {
	m := make(map[string]map[string]string)

	m["dashboard"] = make(map[string]string)
	m["dashboard.dev"] = make(map[string]string)

	// General properties for all environments
	m["dashboard"]["prop1"] = "A"
	m["dashboard"]["prop2"] = "A"

	// Specific properties for dev environment. Note that "prop2" is missing here.
	m["dashboard.dev"]["prop1"] = "B"

	properties := filterProperties("dashboard.dev", m)

	fmt.Println(properties)
	assert.Check(t, properties["dashboard.dev"]["prop1"] == "B")
	assert.Check(t, len(properties["dashboard.dev"]) == 1)
}

func TestSkipConfigDeployment(t *testing.T) {

	m := getTestPropertiesWithGroupAndEnvironment()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	skipDeployment := config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test.prod-environment"], SkipConfigDeploymentParameter)
	m["test.production"][SkipConfigDeploymentParameter] = "true"
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	skipDeployment = config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test.production"], SkipConfigDeploymentParameter)
	m["test"][SkipConfigDeploymentParameter] = "true"
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	skipDeployment = config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test"], SkipConfigDeploymentParameter)
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	skipDeployment = config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, false, skipDeployment)
}

// Test getting object name for environment
// considering environment and group overrides
func TestGetObjectNameForEnvironment(t *testing.T) {

	m := getTestPropertiesWithGroupAndEnvironment()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	productionResult, err := config.GetObjectNameForEnvironment(testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Prod environment config name", productionResult)

	// remove name parameter from test.prod-environment
	// and check if group `name` parameter is set
	delete(m["test.prod-environment"], "name")
	productionResult, err = config.GetObjectNameForEnvironment(testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Production config name", productionResult)

	// remove name parameter from test.production
	// and check if group `name` parameter is set
	delete(m["test.production"], "name")
	productionResult, err = config.GetObjectNameForEnvironment(testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Config name", productionResult)

	// remove name parameter from test config
	// this test should fail as no name parameter is defined
	delete(m["test"], "name")
	productionResult, err = config.GetObjectNameForEnvironment(testProductionEnvironment, make(map[string]api.DynatraceEntity))

	expected := util.ReplacePathSeparators("could not find name property in config testproject/management-zone/test, please make sure `name` is defined and not empty")
	assert.Error(t, err, expected)
}

func getTestTemplate(t *testing.T) util.Template {
	template, e := util.NewTemplateFromString("test", testTemplate)
	assert.NilError(t, e)
	return template
}

func getTestPropertiesWithGroupAndEnvironment() map[string]map[string]string {

	m := make(map[string]map[string]string)

	m["test"] = make(map[string]string)
	m["test"]["name"] = "Config name"
	m["test"]["color"] = "white"
	m["test"]["animalType"] = "rabbit"

	m["test.production"] = make(map[string]string)
	m["test.production"]["name"] = "Production config name"
	m["test.production"]["color"] = "brown"
	m["test.production"]["animalType"] = "dog"

	m["test.prod-environment"] = make(map[string]string)
	m["test.prod-environment"]["name"] = "Prod environment config name"
	m["test.prod-environment"]["color"] = "red"
	m["test.prod-environment"]["animalType"] = "cat"
	m["test.prod-environment"][SkipConfigDeploymentParameter] = "true"

	return m
}

func TestHasDependencyCheck(t *testing.T) {
	prop := make(map[string]map[string]string)
	prop["test"] = make(map[string]string)
	prop["test"]["name"] = "A name"
	prop["test"]["somethingelse"] = util.ReplacePathSeparators("testproject/management-zone/other.id")
	temp, e := util.NewTemplateFromString("test", "{{.name}}{{.somethingelse}}")
	assert.NilError(t, e)

	config := newConfig("test", "testproject", temp, prop, testManagementZoneApi, "test.json")

	otherConfig := newConfig("other", "testproject", temp, make(map[string]map[string]string), testManagementZoneApi, "other.json")

	assert.Equal(t, true, config.HasDependencyOn(otherConfig))
}

func TestHasDependencyWithMultipleDependenciesCheck(t *testing.T) {
	prop := make(map[string]map[string]string)
	prop["test"] = make(map[string]string)
	prop["test"]["name"] = "A name"

	prop["test"]["someDependency"] = "management-zone/not-existing-dep.name"
	prop["test"]["somethingelse"] = util.ReplacePathSeparators("management-zone/other.id")
	temp, e := util.NewTemplateFromString("test", "{{.name}}{{.somethingelse}}")
	assert.NilError(t, e)

	config := newConfig("test", "testproject", temp, prop, testManagementZoneApi, "test.json")

	otherConfig := newConfig("other", "testproject", temp, make(map[string]map[string]string), testManagementZoneApi, "other.json")

	assert.Equal(t, true, config.HasDependencyOn(otherConfig))
}

func TestParseDependencyWithAbsolutePath(t *testing.T) {

	prop := make(map[string]map[string]string)
	templ := getTestTemplate(t)

	config := createConfigForTest("test", "testproject", templ, prop, testManagementZoneApi, "")

	managementZonePath := util.ReplacePathSeparators("infrastructure/management-zone/zone")

	dynatraceEntity := api.DynatraceEntity{
		Description: "bla",
		Name:        "Test Management Zone",
		Id:          managementZonePath,
	}
	dict := make(map[string]api.DynatraceEntity)
	dict[managementZonePath] = dynatraceEntity

	managementZoneId, err := config.parseDependency(string(os.PathSeparator)+managementZonePath+".name", dict)
	assert.NilError(t, err)
	assert.Equal(t, "Test Management Zone", managementZoneId)
}

func TestParseDependencyWithRelativePath(t *testing.T) {

	prop := make(map[string]map[string]string)
	templ := getTestTemplate(t)

	config := createConfigForTest("test", "testproject", templ, prop, testManagementZoneApi, "")

	dynatraceEntity := api.DynatraceEntity{
		Description: "bla",
		Name:        "Test Management Zone",
		Id:          "zone",
	}
	dict := make(map[string]api.DynatraceEntity)
	dict["infrastructure/management-zone/zone"] = dynatraceEntity

	managementZoneId, err := config.parseDependency("infrastructure/management-zone/zone.id", dict)
	assert.NilError(t, err)
	assert.Equal(t, "zone", managementZoneId)
}
