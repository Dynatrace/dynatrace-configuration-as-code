//go:build integration

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

package v2

import (
	"encoding/json"
	"fmt"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/client"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/google/uuid"
	"gotest.tools/assert"
	"net/http"
	"os"
	"testing"
)

func TestNonUniqueNameUpserts(t *testing.T) {
	testSuffix := generateTestSuffix("NonUniqueName")

	t.Cleanup(func() {
		cleanupIntegrationTest(
			t,
			manifest.Manifest{
				Projects: nil,
				Environments: map[string]manifest.EnvironmentDefinition{
					"test": manifest.NewEnvironmentDefinition("test", manifest.UrlDefinition{Type: manifest.EnvironmentUrlType, Value: "URL_ENVIRONMENT_1"}, "default", &manifest.EnvironmentVariableToken{EnvironmentVariableName: "TOKEN_ENVIRONMENT_1"}),
				},
			},
			"test",
			testSuffix,
		)
	})

	url := os.Getenv("URL_ENVIRONMENT_1")
	token := os.Getenv("TOKEN_ENVIRONMENT_1")

	httpClient := &http.Client{}

	c, err := client.NewDynatraceClient(url, token, client.WithHTTPClient(httpClient))
	assert.NilError(t, err)

	a := api.NewApis()["alerting-profile"]
	assert.Assert(t, a.IsNonUniqueNameApi())

	name := "TestObject_" + testSuffix
	payload := []byte(fmt.Sprintf(`{ "displayName": "%s", "rules": [] }`, name))

	// ensure blank slate start
	existing := getConfigsOfName(t, c, a, name)
	assert.Assert(t, len(existing) == 0, "Test requires no pre-existing configs of name %q but found %d", name, len(existing))

	// create initial object of unknown UUID via direct PUT
	randomUUID := getRandomUUID(t)
	createObjectViaDirectPut(t, httpClient, url, a, token, randomUUID, payload)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// 1. if only one config of non-unique-name exist it MUST be updated
	expectedUUID := util.GenerateUuidFromConfigId("test_project", name)
	e, err := c.UpsertByNonUniqueNameAndId(a, expectedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, randomUUID, "expected existing single config %d to be updated, but reply UUID was", randomUUID, e.Id)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// generate additional config
	additionalUUID := getRandomUUID(t)
	createObjectViaDirectPut(t, httpClient, url, a, token, additionalUUID, payload)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 2, "Expected two configs of name %q but found %d", name, len(existing))

	// 2. if several configs of non-unique-name exist an additional config with monaco controlled UUID is created
	assert.NilError(t, err)
	e, err = c.UpsertByNonUniqueNameAndId(a, expectedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, expectedUUID)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))

	// 3. if several configs of non-unique-name exist and one with known monaco-controlled UUID is found that MUST be updated
	assert.NilError(t, err)
	e, err = c.UpsertByNonUniqueNameAndId(a, expectedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, expectedUUID)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))
}

func getConfigsOfName(t *testing.T, c client.Client, a api.Api, name string) []api.Value {
	var existingEntities []api.Value
	entities, err := c.List(a)
	assert.NilError(t, err)
	for _, e := range entities {
		if e.Name == name {
			existingEntities = append(existingEntities, e)
		}
	}
	return existingEntities
}

func getRandomUUID(t *testing.T) string {
	uuid, err := uuid.NewUUID()
	assert.NilError(t, err)
	return uuid.String()
}

func createObjectViaDirectPut(t *testing.T, client *http.Client, url string, a api.Api, apiToken string, id string, payload []byte) {
	res, err := rest.Put(client, a.GetUrl(url)+"/"+id, payload, apiToken)
	assert.NilError(t, err)
	assert.Assert(t, res.StatusCode >= 200 && res.StatusCode < 300)

	var dtEntity api.DynatraceEntity
	err = json.Unmarshal(res.Body, &dtEntity)
	assert.NilError(t, err)

	assert.Equal(t, dtEntity.Id, id)
}
