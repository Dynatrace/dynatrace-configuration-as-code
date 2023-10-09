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
	"context"
	"encoding/json"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	uuid2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/auth"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/google/uuid"
	"gotest.tools/assert"
	"os"
	"strings"
	"testing"
)

func TestNonUniqueNameUpserts(t *testing.T) {
	testSuffix := integrationtest.GenerateTestSuffix(t, "NonUniqueName")

	url := os.Getenv("URL_ENVIRONMENT_1")
	token := os.Getenv("TOKEN_ENVIRONMENT_1")

	name := "TestObject_" + testSuffix
	firstExistingObjectUUID := getRandomUUID(t)
	monacoGeneratedUUID := uuid2.GenerateUUIDFromConfigId("test_project", name)
	secondExistingObjectUUID := getRandomUUID(t)

	httpClient := rest.NewRestClient(auth.NewTokenAuthClient(token), nil, rest.CreateRateLimitStrategy())
	c, err := dtclient.NewClassicClient(url, httpClient)
	assert.NilError(t, err)

	a := api.NewAPIs()["alerting-profile"]
	assert.Assert(t, a.NonUniqueName)

	t.Cleanup(func() {
		for _, id := range []string{firstExistingObjectUUID, secondExistingObjectUUID, monacoGeneratedUUID} {
			if err := c.DeleteConfigById(a, id); err != nil {
				t.Log("failed to cleanup test config with ID: ", id)
			}
		}
	})

	payload := []byte(fmt.Sprintf(`{ "displayName": "%s", "rules": [] }`, name))

	// ensure blank slate start
	existing := getConfigsOfName(t, c, a, name)
	assert.Assert(t, len(existing) == 0, "Test requires no pre-existing configs of name %q but found %d", name, len(existing))

	// create initial object of unknown UUID via direct PUT
	createObjectViaDirectPut(t, httpClient, url, a, firstExistingObjectUUID, payload)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// 1. if only one config of non-unique-name exist it MUST be updated
	e, err := c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, firstExistingObjectUUID, "expected existing single config %d to be updated, but reply UUID was", firstExistingObjectUUID, e.Id)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// generate additional config
	createObjectViaDirectPut(t, httpClient, url, a, secondExistingObjectUUID, payload)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 2, "Expected two configs of name %q but found %d", name, len(existing))

	// 2. if several configs of non-unique-name exist an additional config with monaco controlled UUID is created
	assert.NilError(t, err)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))

	// 3. if several configs of non-unique-name exist and one with known monaco-controlled UUID is found that MUST be updated
	assert.NilError(t, err)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload)
	assert.NilError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID)
	assert.Assert(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))
}

func getConfigsOfName(t *testing.T, c dtclient.Client, a api.API, name string) []dtclient.Value {
	var existingEntities []dtclient.Value
	entities, err := c.ListConfigs(context.TODO(), a)
	assert.NilError(t, err)
	for _, e := range entities {
		if e.Name == name {
			existingEntities = append(existingEntities, e)
		}
	}
	return existingEntities
}

func getRandomUUID(t *testing.T) string {
	id, err := uuid.NewUUID()
	assert.NilError(t, err)
	return id.String()
}

func createObjectViaDirectPut(t *testing.T, c *rest.Client, url string, a api.API, id string, payload []byte) {
	url = strings.TrimSuffix(url, "/")
	res, err := c.Put(context.TODO(), a.CreateURL(url)+"/"+id, payload)
	assert.NilError(t, err)
	assert.Assert(t, res.StatusCode >= 200 && res.StatusCode < 300, "Expected status code to be within [200, 299], but was %d. Response-body: %v", res.StatusCode, string(res.Body))

	var dtEntity dtclient.DynatraceEntity
	err = json.Unmarshal(res.Body, &dtEntity)
	assert.NilError(t, err)

	assert.Equal(t, dtEntity.Id, id)
}
