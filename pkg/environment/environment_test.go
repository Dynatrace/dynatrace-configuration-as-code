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

package environment

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/assert"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
)

const testYamlEnvironment = `
development:
    - name: "Dev"
    - env-url: "https://url/to/dev/environment"
    - env-token-name: "DEV"
hardening:
    - name: "Hardening"
    - env-url: "https://url/to/hardening/environment"
    - env-token-name: "HARDENING"
production.prod-environment:
    - name: "prod-environment"
    - env-url: "https://url/to/production/environment"
    - env-token-name: "PRODUCTION"
`

const testYamlEnvironmentWithGroups = `
.dev:
    - name: "Dev"
    - env-url: "https://url/to/dev/environment"
    - env-token-name: "DEV"
stage.:
    - name: "Stage"
    - env-url: "https://url/to/stage/environment"
    - env-token-name: "STAGE"
test.group.hardening:
    - name: "Hardening"
    - env-url: "https://url/to/hardening/environment"
    - env-token-name: "HARDENING"
production.prod-environment:
    - name: "prod-environment"
    - env-url: "https://url/to/production/environment"
    - env-token-name: "PRODUCTION"
`

const testYamlEnvironmentSameIds = `
development.myenvironment:
  - name: "myDevEnvironment"
  - env-url: "https://myenvironment1.dynatrace.com"
  - env-token-name: "MYENV_TOKEN"
production.myenvironment:
  - name: "myProdEnvironment"
  - env-url: "https://myenvironment2.dynatrace.com"
  - env-token-name: "MYENV_TOKEN"
`

const testYamlEnvironmentWithNewPropertyFormat = `
development:
    - name: "Dev"
    - env-url: "{{.Env.URL}}"
    - env-token-name: "DEV"
`

var testDevEnvironment = NewEnvironment("development", "Dev", "", "https://url/to/dev/environment", "DEV")
var testHardeningEnvironment = NewEnvironment("hardening", "Hardening", "", "https://url/to/hardening/environment", "HARDENING")
var testProductionEnvironment = NewEnvironment("prod-environment", "prod-environment", "production", "https://url/to/production/environment", "PRODUCTION")
var testTrailingSlashEnvironment = NewEnvironment("trailing-slash-environment", "trailing-slash-environment", "", "https://url/to/production/environment/", "TRAILINGSLASH")

func TestShouldParseYaml(t *testing.T) {

	e, result := util.UnmarshalYaml(testYamlEnvironment, "test-yaml")
	assert.NilError(t, e)

	environments, errorList := NewEnvironments(result)

	assert.Check(t, len(errorList) == 0)
	assert.Check(t, len(environments) == 3)

	dev := environments["development"]
	hardening := environments["hardening"]
	production := environments["prod-environment"]

	assert.Check(t, dev != nil)
	assert.Check(t, hardening != nil)
	assert.Check(t, production != nil)

	assert.DeepEqual(t, dev, testDevEnvironment, cmp.AllowUnexported(environmentImpl{}))
	assert.DeepEqual(t, hardening, testHardeningEnvironment, cmp.AllowUnexported(environmentImpl{}))
	assert.DeepEqual(t, production, testProductionEnvironment, cmp.AllowUnexported(environmentImpl{}))
}

func TestParsingEnvironmentsWithMultipleGroups(t *testing.T) {
	e, result := util.UnmarshalYaml(testYamlEnvironmentWithGroups, "test-yaml")
	assert.NilError(t, e)

	environments, errorList := NewEnvironments(result)
	assert.Check(t, len(errorList) == 3)
	assert.Check(t, len(environments) == 1)

	production := environments["prod-environment"]
	assert.Check(t, production != nil)
	assert.DeepEqual(t, production, testProductionEnvironment, cmp.AllowUnexported(environmentImpl{}))

}

func TestParsingEnvironmentsWithSameIds(t *testing.T) {
	e, result := util.UnmarshalYaml(testYamlEnvironmentSameIds, "test-yaml")
	assert.NilError(t, e)

	environments, errorList := NewEnvironments(result)
	assert.Check(t, len(errorList) == 1)
	assert.Check(t, len(environments) == 1)

	myenvironment := environments["myenvironment"]
	assert.Check(t, myenvironment != nil)
}

func TestTokenAvailableOnGetterCall(t *testing.T) {
	testTokenAvailableOnGetterCall(t, testYamlEnvironment)
}

func TestTokenNotAvailableOnGetterCall(t *testing.T) {
	testTokenNotAvailableOnGetterCall(t, testYamlEnvironment)
}

func testTokenAvailableOnGetterCall(t *testing.T, environmentYamlContent string) {

	e, devEnvironment := setupEnvironment(t, environmentYamlContent, "development")

	util.SetEnv(t, "DEV", "1234")
	token, e := devEnvironment.GetToken()

	assert.NilError(t, e)
	assert.Equal(t, "1234", token)

	util.UnsetEnv(t, "DEV")
}

func testTokenNotAvailableOnGetterCall(t *testing.T, environmentYamlContent string) {

	e, devEnvironment := setupEnvironment(t, environmentYamlContent, "development")
	_, e = devEnvironment.GetToken()

	assert.Error(t, e, "environment variable DEV not found")
}

func TestUrlAvailableWithTemplating(t *testing.T) {

	util.SetEnv(t, "URL", "1234")
	e, devEnvironment := setupEnvironment(t, testYamlEnvironmentWithNewPropertyFormat, "development")

	assert.NilError(t, e)
	assert.Equal(t, "1234", devEnvironment.GetEnvironmentUrl())

	util.UnsetEnv(t, "URL")
}

func TestTokenNotAvailableOnGetterCallWithTemplating(t *testing.T) {

	e, _ := util.UnmarshalYaml(testYamlEnvironmentWithNewPropertyFormat, "test-yaml")
	assert.ErrorContains(t, e, "map has no entry for key \"URL\"")
}

func TestTrailingSlashTrimmedFromEnvironmentURL(t *testing.T) {
	envURL := testTrailingSlashEnvironment.GetEnvironmentUrl()
	last_char := envURL[len(envURL)-1:]

	if last_char == "/" {
		t.Errorf("Env URL is: %s; Last Char is: %s. Expected last character NOT to be a trailing slash.", envURL, last_char)
	}
}

func setupEnvironment(t *testing.T, environmentYamlContent string, environmentOfInterest string) (error, Environment) {

	e, result := util.UnmarshalYaml(environmentYamlContent, "test-yaml")
	assert.NilError(t, e)

	environments, errorList := NewEnvironments(result)
	assert.Check(t, len(errorList) == 0)

	devEnvironment := environments[environmentOfInterest]
	assert.Check(t, devEnvironment != nil)

	return e, devEnvironment
}
