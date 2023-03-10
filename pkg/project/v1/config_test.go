//go:build unit

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"gotest.tools/assert"
)

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

func TestHasDependencyCheck(t *testing.T) {
	testManagementZoneApi := api.API{ID: "management-zone", URLPath: "/api/config/v1/managementZones"}
	prop := make(map[string]map[string]string)
	prop["test"] = make(map[string]string)
	prop["test"]["name"] = "A name"
	prop["test"]["somethingelse"] = files.ReplacePathSeparators("testproject/management-zone/other.id")
	temp, e := template.NewTemplateFromString("test", "{{.name}}{{.somethingelse}}")
	assert.NilError(t, e)

	config := newConfigWithTemplate("test", "testproject", temp, prop, testManagementZoneApi, "test.json")

	otherConfig := newConfigWithTemplate("other", "testproject", temp, make(map[string]map[string]string), testManagementZoneApi, "other.json")

	assert.Equal(t, true, config.HasDependencyOn(otherConfig))
}

func TestHasDependencyWithMultipleDependenciesCheck(t *testing.T) {
	testManagementZoneApi := api.API{ID: "management-zone", URLPath: "/api/config/v1/managementZones"}

	prop := make(map[string]map[string]string)
	prop["test"] = make(map[string]string)
	prop["test"]["name"] = "A name"

	prop["test"]["someDependency"] = "management-zone/not-existing-dep.name"
	prop["test"]["somethingelse"] = files.ReplacePathSeparators("management-zone/other.id")
	temp, e := template.NewTemplateFromString("test", "{{.name}}{{.somethingelse}}")
	assert.NilError(t, e)

	config := newConfigWithTemplate("test", "testproject", temp, prop, testManagementZoneApi, "test.json")

	otherConfig := newConfigWithTemplate("other", "testproject", temp, make(map[string]map[string]string), testManagementZoneApi, "other.json")

	assert.Equal(t, true, config.HasDependencyOn(otherConfig))
}
