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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	corerest "github.com/dynatrace/dynatrace-configuration-as-code-core/api/rest"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	uuid2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

// TestNonUniqueNameUpserts asserts the logic of non-unique name configs being updated by name if only a single one is found.
// As this behaviour can be unwanted if a project actually contains several configs of the same name (they'll all just update one object)
// it can also be deactivated - which is tested by TestNonUniqueNameUpserts_InactiveUpdateByName.
func TestNonUniqueNameUpserts(t *testing.T) {
	testSuffix := integrationtest.GenerateTestSuffix(t, "NonUniqueName")

	url := os.Getenv("URL_ENVIRONMENT_1")
	token := os.Getenv("TOKEN_ENVIRONMENT_1")

	name := "TestObject_" + testSuffix
	firstExistingObjectUUID := getRandomUUID(t)
	monacoGeneratedUUID := uuid2.GenerateUUIDFromConfigId("test_project", name)
	secondExistingObjectUUID := getRandomUUID(t)

	factory := clients.Factory().
		WithAccessToken(token).WithClassicURL(url)

	httpClient, err := factory.CreateClassicClient()
	require.NoError(t, err)

	c, err := dtclient.NewClassicClient(httpClient)
	assert.NoError(t, err)

	a := api.NewAPIs()["alerting-profile"]
	assert.True(t, a.NonUniqueName)

	t.Cleanup(func() {
		for _, id := range []string{firstExistingObjectUUID, secondExistingObjectUUID, monacoGeneratedUUID} {
			if err := c.DeleteConfigById(context.TODO(), a, id); err != nil {
				t.Log("failed to cleanup test config with ID: ", id)
			}
		}
	})

	payload := []byte(fmt.Sprintf(`{ "displayName": "%s", "rules": [] }`, name))

	// ensure blank slate start
	existing := getConfigsOfName(t, c, a, name)
	assert.True(t, len(existing) == 0, "Test requires no pre-existing configs of name %q but found %d", name, len(existing))

	// create initial object of unknown UUID via direct PUT
	createObjectViaDirectPut(t, httpClient, url, a, firstExistingObjectUUID, payload)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// 1. if only one config of non-unique-name exist it MUST be updated
	e, err := c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, firstExistingObjectUUID, "expected existing single config %d to be updated, but reply UUID was", firstExistingObjectUUID, e.Id)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// 1.1. Deploying another config of the same name is also just an update (unwanted behaviour if a project re-uses names)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, uuid2.GenerateUUIDFromConfigId("test_project", "other-config"), name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, firstExistingObjectUUID, "expected existing single config %d to be updated, but reply UUID was", firstExistingObjectUUID, e.Id)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// generate additional config
	createObjectViaDirectPut(t, httpClient, url, a, secondExistingObjectUUID, payload)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 2, "Expected two configs of name %q but found %d", name, len(existing))

	// 2. if several configs of non-unique-name exist an additional config with monaco controlled UUID is created
	assert.NoError(t, err)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))

	// 3. if several configs of non-unique-name exist and one with known monaco-controlled UUID is found that MUST be updated
	assert.NoError(t, err)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected three configs of name %q but found %d", name, len(existing))
}

// TestNonUniqueNameUpserts_InactiveUpdateByName asserts that the logic to update single non-unique name configs can be
// deactivated. For the base behaviour see TestNonUniqueNameUpserts.
func TestNonUniqueNameUpserts_InactiveUpdateByName(t *testing.T) {

	t.Setenv(featureflags.Permanent[featureflags.UpdateNonUniqueByNameIfSingleOneExists].EnvName(), "false")

	testSuffix := integrationtest.GenerateTestSuffix(t, "NonUniqueName")

	url := os.Getenv("URL_ENVIRONMENT_1")
	token := os.Getenv("TOKEN_ENVIRONMENT_1")

	name := "TestObject_" + testSuffix
	firstExistingObjectUUID := getRandomUUID(t)
	monacoGeneratedUUID := uuid2.GenerateUUIDFromConfigId("test_project", name)
	otherMonacoGeneratedUUID := uuid2.GenerateUUIDFromConfigId("test_project", "other-config_"+testSuffix)
	secondExistingObjectUUID := getRandomUUID(t)

	classicClient, err := clients.Factory().WithClassicURL(url).WithAccessToken(token).CreateClassicClient()
	require.NoError(t, err)

	c, err := dtclient.NewClassicClient(classicClient)
	assert.NoError(t, err)

	a := api.NewAPIs()["alerting-profile"]
	assert.True(t, a.NonUniqueName)

	t.Cleanup(func() {
		for _, id := range []string{firstExistingObjectUUID, secondExistingObjectUUID, monacoGeneratedUUID, otherMonacoGeneratedUUID} {
			if err := c.DeleteConfigById(context.TODO(), a, id); err != nil {
				t.Log("failed to cleanup test config with ID: ", id)
			}
		}
	})

	payload := []byte(fmt.Sprintf(`{ "displayName": "%s", "rules": [] }`, name))

	// ensure blank slate start
	existing := getConfigsOfName(t, c, a, name)
	assert.True(t, len(existing) == 0, "Test requires no pre-existing configs of name %q but found %d", name, len(existing))

	// create initial object of unknown UUID via direct PUT
	createObjectViaDirectPut(t, classicClient, url, a, firstExistingObjectUUID, payload)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 1, "Expected single configs of name %q but found %d", name, len(existing))

	// 1. if only one config of non-unique-name exist an additional one is still create (update feature OFF)
	e, err := c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID, "expected existing single config %d to be updated, but reply UUID was", firstExistingObjectUUID, e.Id)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 2, "Expected single configs of name %q but found %d", name, len(existing))

	// 2. Deploying another config of the same name is also just an update (unwanted behaviour if a project re-uses names)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, otherMonacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, otherMonacoGeneratedUUID, "expected existing single config %d to be updated, but reply UUID was", firstExistingObjectUUID, e.Id)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 3, "Expected single configs of name %q but found %d", name, len(existing))

	// generate additional config
	createObjectViaDirectPut(t, classicClient, url, a, secondExistingObjectUUID, payload)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 4, "Expected two configs of name %q but found %d", name, len(existing))

	// 3. if several configs of non-unique-name exist and one with known monaco-controlled UUID is found that MUST be updated
	assert.NoError(t, err)
	e, err = c.UpsertConfigByNonUniqueNameAndId(context.TODO(), a, monacoGeneratedUUID, name, payload, false)
	assert.NoError(t, err)
	assert.Equal(t, e.Id, monacoGeneratedUUID)
	assert.True(t, len(getConfigsOfName(t, c, a, name)) == 4, "Expected three configs of name %q but found %d", name, len(existing))
}

func getConfigsOfName(t *testing.T, c client.ConfigClient, a api.API, name string) []dtclient.Value {
	var existingEntities []dtclient.Value
	entities, err := c.ListConfigs(context.TODO(), a)
	assert.NoError(t, err)
	for _, e := range entities {
		if e.Name == name {
			existingEntities = append(existingEntities, e)
		}
	}
	return existingEntities
}

func getRandomUUID(t *testing.T) string {
	id, err := uuid.NewUUID()
	assert.NoError(t, err)
	return id.String()
}

func createObjectViaDirectPut(t *testing.T, c *corerest.Client, url string, a api.API, id string, payload []byte) {
	url = strings.TrimSuffix(url, "/")
	res, err := coreapi.AsResponseOrError(c.PUT(context.TODO(), a.URLPath+"/"+id, bytes.NewReader(payload), corerest.RequestOptions{}))
	require.NoError(t, err)

	var dtEntity dtclient.DynatraceEntity
	err = json.Unmarshal(res.Data, &dtEntity)
	assert.NoError(t, err)

	assert.Equal(t, dtEntity.Id, id)
}
