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

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/environment"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"gotest.tools/assert"
)

const testTemplate = `{"msg": "Follow the {{.color}} {{.animalType}}"}`
const testTemplateWithDependency = `{"msg": "Follow the {{.color}} {{.animalType}} with {{ .dep }}"}`
const testTemplateWithEnvVar = `{"msg": "Follow the {{.color}} {{ .Env.ANIMAL }}"}`
const testHostAutoUpdateTemplate = `{"updateWindows": { "windows": ["window"] }}`
const testHostAutoUpdateTemplateWithEmptyWindows = `{"updateWindows": { "windows": [] }}`

var testDevEnvironment = environment.NewEnvironment("development", "Dev", "", "https://url/to/dev/environment", "DEV")
var testHardeningEnvironment = environment.NewEnvironment("hardening", "Hardening", "", "https://url/to/hardening/environment", "HARDENING")
var testProductionEnvironment = environment.NewEnvironment("prod-environment", "prod-environment", "production", "https://url/to/production/environment", "PRODUCTION")
var testManagementZoneApi = api.NewStandardApi("management-zone", "/api/config/v1/managementZones", false)

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

func TestGetConfigStringWithEnvironmentOverride(t *testing.T) {

	m := getTestProperties()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	devResult, err := getConfigForEnvironmentAsMap(config, testDevEnvironment, make(map[string]api.DynatraceEntity))

	assert.NilError(t, err)
	assert.Equal(t, "Follow the black squid", devResult["msg"])
}

func TestGetConfigStringNoEnvironmentOverride(t *testing.T) {

	m := getTestProperties()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	hardeningResult, err := getConfigForEnvironmentAsMap(config, testHardeningEnvironment, make(map[string]api.DynatraceEntity))

	assert.NilError(t, err)
	assert.Equal(t, "Follow the white rabbit", hardeningResult["msg"])
}

func TestGetConfigString(t *testing.T) {

	m := getTestProperties()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	devResult, devErr := getConfigForEnvironmentAsMap(config, testDevEnvironment, make(map[string]api.DynatraceEntity))
	hardeningResult, hardeningErr := getConfigForEnvironmentAsMap(config, testHardeningEnvironment, make(map[string]api.DynatraceEntity))

	assert.NilError(t, devErr)
	assert.NilError(t, hardeningErr)
	assert.Equal(t, "Follow the black squid", devResult["msg"])
	assert.Equal(t, "Follow the white rabbit", hardeningResult["msg"])
}

// test GetConfigForEnvironment if environment group is defined
// it should return `test.production` group values of getTestProperties
func TestGetConfigWithGroupOverride(t *testing.T) {

	m := getTestProperties()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	productionResult, err := getConfigForEnvironmentAsMap(config, testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Follow the brown dog", productionResult["msg"])
}

// test GetConfigForEnvironment if environment group is defined
// it should return `test.production` group values of getTestProperties
func TestGetConfigWithGroupOverrideAndDependency(t *testing.T) {

	m := getTestProperties()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	productionResult, err := getConfigForEnvironmentAsMap(config, testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Follow the brown dog", productionResult["msg"])
}

// Testing dependencies resolution within group and env overrides
func TestGetConfigWithGroupAndEnvironmentOverride(t *testing.T) {

	managementZonePath := util.ReplacePathSeparators("infrastructure/management-zone/zone")

	dynatraceEntity := api.DynatraceEntity{
		Description: "bla",
		Name:        "Test Management Zone",
		Id:          managementZonePath,
	}

	dict := make(map[string]api.DynatraceEntity)
	dict[managementZonePath] = dynatraceEntity

	m := getTestPropertiesWithGroupAndEnvironment()
	m["test.production"]["dep"] = "infrastructure/management-zone/zone.name"
	templ := getTestTemplateWithDependency(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	productionResult, err := getConfigForEnvironmentAsMap(config, testProductionEnvironment, dict)
	assert.NilError(t, err)
	assert.Equal(t, "Follow the red cat with Test Management Zone", productionResult["msg"])

	m["test.prod-environment"]["dep2"] = "infrastructure/management-zone/zone.name"
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	productionResult, err = getConfigForEnvironmentAsMap(config, testProductionEnvironment, dict)
	assert.NilError(t, err)
	assert.Equal(t, "Follow the red cat with Test Management Zone", productionResult["msg"])
}

// Test combining parameters
// If there are different parameters defined for group and environment, they should be merged
func TestGetConfigWithMergingGroupAndEnvironmentOverrides(t *testing.T) {
	m := getTestPropertiesWithGroupAndEnvironment()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	// remove color parameter from `test.prod-environment`
	// `test.production.color` parameter should be taken instead
	delete(m["test.prod-environment"], "color")
	productionResult, err := getConfigForEnvironmentAsMap(config, testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Follow the brown cat", productionResult["msg"])

	// removing whole `test.prod-environment` config section
	// only `test.production` parameters should be considered
	delete(m, "test.prod-environment")
	productionResult, err = getConfigForEnvironmentAsMap(config, testProductionEnvironment, make(map[string]api.DynatraceEntity))
	assert.NilError(t, err)
	assert.Equal(t, "Follow the brown dog", productionResult["msg"])
}

func TestSkipConfigDeployment(t *testing.T) {

	m := getTestPropertiesWithGroupAndEnvironment()
	templ := getTestTemplate(t)
	config := newConfig("test", "testproject", templ, m, testManagementZoneApi, "")

	skipDeployment := config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test.prod-environment"], skipConfigDeploymentParameter)
	m["test.production"][skipConfigDeploymentParameter] = "true"
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	skipDeployment = config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test.production"], skipConfigDeploymentParameter)
	m["test"][skipConfigDeploymentParameter] = "true"
	config = newConfig("test", "testproject", templ, m, testManagementZoneApi, "")
	skipDeployment = config.IsSkipDeployment(testProductionEnvironment)
	assert.Equal(t, true, skipDeployment)

	delete(m["test"], skipConfigDeploymentParameter)
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

	expected := util.ReplacePathSeparators("could not find name property in config testproject/management-zone/test, please make sure `name` is defined")
	assert.Error(t, err, expected)
}

func getTestTemplate(t *testing.T) util.Template {
	template, e := util.NewTemplateFromString("test", testTemplate)
	assert.NilError(t, e)
	return template
}

func getTestTemplateWithDependency(t *testing.T) util.Template {
	template, e := util.NewTemplateFromString("test", testTemplateWithDependency)
	assert.NilError(t, e)
	return template
}

func getTestTemplateWithEnvVars(t *testing.T) util.Template {
	template, e := util.NewTemplateFromString("test", testTemplateWithEnvVar)
	assert.NilError(t, e)
	return template
}

func getTestProperties() map[string]map[string]string {

	m := make(map[string]map[string]string)

	m["test"] = make(map[string]string)
	m["test"]["color"] = "white"
	m["test"]["animalType"] = "rabbit"

	m["test.development"] = make(map[string]string)
	m["test.development"]["color"] = "black"
	m["test.development"]["animalType"] = "squid"

	m["test.production"] = make(map[string]string)
	m["test.production"]["color"] = "brown"
	m["test.production"]["animalType"] = "dog"

	return m
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
	m["test.prod-environment"][skipConfigDeploymentParameter] = "true"

	return m
}

func TestReplaceDependency(t *testing.T) {

	entity1 := api.DynatraceEntity{Id: "0815", Name: "MyCustomObj"}
	entity2 := api.DynatraceEntity{Id: "asdf", Name: "MySuperObj"}

	dict := make(map[string]api.DynatraceEntity)
	dict["Foo"] = entity1
	dict["Bar"] = entity2

	data := make(map[string]map[string]string)
	data["obj"] = make(map[string]string)

	data["obj"]["k1"] = "value"
	data["obj"]["k2"] = "Bar.id"
	data["obj"]["k3"] = "Foo.name"

	config := configImpl{}
	data, err := config.replaceDependencies(data, dict)
	assert.NilError(t, err)
	assert.Equal(t, "value", data["obj"]["k1"])
	assert.Equal(t, "asdf", data["obj"]["k2"])
	assert.Equal(t, "MyCustomObj", data["obj"]["k3"])
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

func TestMeIdRegex(t *testing.T) {
	assert.Check(t, isMeId("HOST_GROUP-95BEC188F318D09C"))
	assert.Check(t, isMeId("APPLICATION-95BEC188F318D09C"))
	assert.Check(t, isMeId("SERVICE-95BEC188F318D09C"))
	assert.Check(t, !isMeId("TOO_SHORT-95BEC188F318D09"))
	assert.Check(t, !isMeId("meId"))
}

func TestGetMeIdProperties(t *testing.T) {

	prop := make(map[string]map[string]string)
	prop["test.development"] = make(map[string]string)
	prop["test.development"]["app1"] = "APPLICATION-95BEC188F318D09C"
	prop["test.development"]["service1"] = "SERVICE-95BEC188F318D09C"
	prop["test.development"]["service2"] = "noMe"
	prop["test2.development"] = make(map[string]string)
	prop["test2.development"]["app1"] = "NOT_AN_APP-1234"
	prop["test3"] = make(map[string]string)
	prop["test3"]["app1"] = "APPLICATION-95BEC188F318D09C"

	config := configImpl{
		properties: prop,
	}

	meIdsOfEnvironment := config.GetMeIdsOfEnvironment(testDevEnvironment)

	assert.Check(t, len(meIdsOfEnvironment) == 1)

	expected := make(map[string]map[string]string)
	expected["test.development"] = make(map[string]string)
	expected["test.development"]["app1"] = "APPLICATION-95BEC188F318D09C"
	expected["test.development"]["service1"] = "SERVICE-95BEC188F318D09C"

	equal := reflect.DeepEqual(expected, meIdsOfEnvironment)
	assert.Check(t, equal)
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

func TestGetConfigStringWithEnvVar(t *testing.T) {

	templ := getTestTemplateWithEnvVars(t)

	util.SetEnv(t, "ANIMAL", "cow")
	config := newConfig("test", "testproject", templ, getTestProperties(), testManagementZoneApi, "")
	devResult, err := getConfigForEnvironmentAsMap(config, testDevEnvironment, make(map[string]api.DynatraceEntity))

	util.UnsetEnv(t, "ANIMAL")

	assert.NilError(t, err)
	assert.Equal(t, "Follow the black cow", devResult["msg"])
}

func TestGetConfigStringWithEnvVarLeadsToErrorIfEnvVarNotPresent(t *testing.T) {

	templ := getTestTemplateWithEnvVars(t)

	util.UnsetEnv(t, "ANIMAL")
	config := newConfig("test", "testproject", templ, getTestProperties(), testManagementZoneApi, "")
	_, err := config.GetConfigForEnvironment(testDevEnvironment, make(map[string]api.DynatraceEntity))

	assert.ErrorContains(t, err, "map has no entry for key \"ANIMAL\"")
}

func getConfigForEnvironmentAsMap(config Config, env environment.Environment, dict map[string]api.DynatraceEntity) (map[string]interface{}, error) {
	data, err := config.GetConfigForEnvironment(env, dict)

	if err != nil {
		return nil, err
	}

	var result map[string]interface{}

	err = json.Unmarshal(data, &result)

	if err != nil {
		return nil, err
	}

	return result, err
}
