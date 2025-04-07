//go:build integration || cleanup || download_restore || nightly

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

package integrationtest

import "C"
import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	coreapi "github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/graph"
	project "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
)

type entityLookup map[coordinate.Coordinate]entities.ResolvedEntity

func (e entityLookup) GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool) {
	if ent, f := e.GetResolvedEntity(coordinate); f {
		if prop, f := ent.Properties[propertyName]; f {
			return prop, true
		}
	}

	return nil, false
}

func (e entityLookup) GetResolvedEntity(config coordinate.Coordinate) (entities.ResolvedEntity, bool) {
	ent, f := e[config]
	return ent, f
}

func sortEnvironments(environments []project.Environment) (map[string][]config.Config, []error) {
	cfgsPerEnv := make(map[string][]config.Config)
	var errs graph.SortingErrors

	for _, environment := range environments {
		sortedCfgs, err := environment.GetSortedConfigs()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		cfgsPerEnv[environment.Name] = sortedCfgs
	}

	if len(errs) > 0 {
		return map[string][]config.Config{}, errs
	}
	return cfgsPerEnv, nil
}

// AssertAllConfigsAvailability checks all configurations of a given project with given availability.
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificProjects []string, specificEnvironment string, available bool) {
	loadedManifest := LoadManifest(t, fs, manifestPath, specificEnvironment)

	environments := LoadEnvironments(t, fs, manifestPath, loadedManifest)

	envNames := make([]string, 0, len(loadedManifest.Environments))

	for _, env := range loadedManifest.Environments {
		envNames = append(envNames, env.Name)
	}

	checkString := "exist"
	if !available {
		checkString = "do NOT exist"
	}

	projectsToValidate := map[string]struct{}{}
	if len(specificProjects) > 0 {
		log.Info("Asserting configurations from projects: %s %s", specificProjects, checkString)
		for _, p := range specificProjects {
			projectsToValidate[p] = struct{}{}
		}
	} else {
		log.Info("Asserting configurations from all projects %s", checkString)
		for _, p := range environments {
			projectsToValidate[p.Name] = struct{}{}
		}
	}

	for _, environment := range environments {
		sortedConfigs, err := environment.GetSortedConfigs()
		require.NoError(t, err, "sorting configurations failed")

		for _, theConfig := range sortedConfigs {

			env := loadedManifest.Environments[environment.Name]

			clients := CreateDynatraceClients(t, env)

			lookup := entityLookup{}
			coord := theConfig.Coordinate

			if theConfig.Skip {
				lookup[coord] = entities.ResolvedEntity{
					EntityName: coord.ConfigId,
					Coordinate: coord,
					Properties: parameter.Properties{},
					Skip:       true,
				}
				continue
			}

			properties, errs := theConfig.ResolveParameterValues(lookup)
			testutils.FailTestOnAnyError(t, errs, "resolving of parameter values failed")

			properties[config.IdParameter] = "NO REAL ID NEEDED FOR CHECKING AVAILABILITY"

			configName, err := extractConfigName(properties)
			assert.NoError(t, err)

			lookup[coord] = entities.ResolvedEntity{
				EntityName: configName,
				Coordinate: coord,
				Properties: properties,
				Skip:       false,
			}

			apis := api.NewAPIs()
			if _, found := projectsToValidate[coord.Project]; found {
				var foundID string
				switch typ := theConfig.Type.(type) {
				case config.SettingsType:
					foundID = AssertSetting(t, clients.SettingsClient, typ, environment, available, theConfig)
				case config.ClassicApiType:
					assert.NotEmpty(t, configName, "classic API config %v is missing name, can not assert if it exists", theConfig.Coordinate)

					theApi := apis[typ.Api]
					if theApi.HasParent() {

						assert.NotEmpty(t, properties[config.ScopeParameter], "subPathAPI config is missing scope")
						scope, ok := properties[config.ScopeParameter].(string)
						assert.True(t, ok, "scope property could not be resolved to string, but was ", properties[config.ScopeParameter])
						theApi = theApi.ApplyParentObjectID(scope)
					}

					foundID = AssertConfig(t, clients.ConfigClient, theApi, available, theConfig, configName, environment)
				case config.AutomationType:
					if clients.AutClient == nil {
						t.Errorf("can not assert existience of Automtation config %q (%s) because no AutomationClient exists - was the test env not configured as Platform?", theConfig.Coordinate, typ.Resource)
						return
					}
					foundID = AssertAutomation(t, clients.AutClient, env, available, typ.Resource, theConfig)
				case config.BucketType:
					if clients.BucketClient == nil {
						t.Errorf("can not assert existience of Bucket config %q) because no BucketClient exists - was the test env not configured as Platform?", theConfig.Coordinate)
						return
					}
					foundID = AssertBucket(t, clients.BucketClient, env, available, theConfig)
				case config.DocumentType:
					if clients.DocumentClient == nil {
						t.Errorf("can not assert existience of Document config %q) because no DocumentClient exists - was the test env not configured as Platform?", theConfig.Coordinate)
						return
					}
				default:
					t.Errorf("Can not assert config of unknown type %q", theConfig.Coordinate.Type)
				}

				if foundID != "" { // store found IDs of asserted configs to allow resolving references (e.g. to assert Settings or SubPath configs referencing other test configs as scope)
					lookup[coord].Properties[config.IdParameter] = foundID
				}
			}
		}
	}
}

func newContextWithLogConfig(t *testing.T, config config.Config, environment project.Environment) context.Context {
	ctx := context.WithValue(t.Context(), log.CtxKeyCoord{}, config.Coordinate)
	ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: environment.Name, Group: environment.Group})
	return ctx
}

func AssertConfig(t *testing.T, client client.ConfigClient, theApi api.API, shouldBeAvailable bool, config config.Config, name string, environment project.Environment) (id string) {
	ctx := newContextWithLogConfig(t, config, environment)
	configType := config.Coordinate.Type

	var exists bool

	if config.Skip {
		exists, _, _ = client.ExistsWithName(ctx, theApi, name)
		assert.False(t, exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := wait(description, 120, func() bool {
		exists, id, _ = client.ExistsWithName(ctx, theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NoError(t, err)

	if shouldBeAvailable {
		assert.True(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	} else {
		assert.False(t, exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	}

	return id
}

func AssertSetting(t *testing.T, c client.SettingsClient, typ config.SettingsType, environment project.Environment, shouldBeAvailable bool, config config.Config) (id string) {
	ctx := newContextWithLogConfig(t, config, environment)
	expectedExtId, err := idutils.GenerateExternalIDForSettingsObject(config.Coordinate)
	if err != nil {
		t.Errorf("Unable to generate external id: %v", err)
		return
	}

	objects, err := c.List(ctx, typ.SchemaId, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == expectedExtId }})
	assert.NoError(t, err)

	if len(objects) > 1 {
		t.Errorf("Expected a specific Settings Object with externalId %q, but %d are present instead.", expectedExtId, len(objects))
		return
	}

	exists := len(objects) == 1

	if config.Skip {
		assert.False(t, exists, "Skipped Settings Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
		return
	}

	if shouldBeAvailable {
		assert.True(t, exists, "Settings Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
	} else {
		assert.False(t, exists, "Settings Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
	}

	if exists {
		return objects[0].ObjectId
	}
	return ""

}

func AssertPermission(t *testing.T, c client.SettingsClient, objectID string, permissions []dtclient.TypePermissions) {
	resp, err := c.GetPermission(t.Context(), objectID)
	if err != nil {
		if len(permissions) == 0 && coreapi.IsNotFoundError(err) {
			return
		}
		t.Errorf("failed to get permissions. Error: %s", err)
		return
	}
	assert.Equal(t, permissions, resp.Permissions)
}

func AssertAutomation(t *testing.T, c client.AutomationClient, env manifest.EnvironmentDefinition, shouldBeAvailable bool, resource config.AutomationResource, cfg config.Config) (id string) {
	resourceType, err := automationutils.ClientResourceTypeFromConfigType(resource)
	assert.NoError(t, err, "failed to get resource type for: %s", cfg.Coordinate)

	var expectedId string
	if cfg.OriginObjectId != "" {
		expectedId = cfg.OriginObjectId
	} else {
		expectedId = idutils.GenerateUUIDFromCoordinate(cfg.Coordinate)
	}

	_, err = c.Get(t.Context(), resourceType, expectedId)
	exists := err == nil

	if cfg.Skip {
		assert.False(t, exists, "Skipped Automation Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
		return
	}

	if shouldBeAvailable {
		assert.True(t, exists, "Automation Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
	} else {
		assert.False(t, exists, "Automation Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
	}
	return expectedId
}

func AssertBucket(t *testing.T, client client.BucketClient, env manifest.EnvironmentDefinition, available bool, cfg config.Config) (id string) {

	var expectedId string
	if cfg.OriginObjectId != "" {
		expectedId = cfg.OriginObjectId
	} else {
		expectedId = idutils.GenerateBucketName(cfg.Coordinate)
	}

	err := waitForBucketToExist(t.Context(), client, expectedId, 120)

	exists := true
	apiErr := coreapi.APIError{}
	if errors.As(err, &apiErr) {
		if coreapi.IsNotFoundError(apiErr) {
			exists = false
		} else {
			assert.NoError(t, apiErr)
		}
	} else if err != nil {
		assert.NoError(t, err)
	}

	if cfg.Skip {
		assert.Falsef(t, exists, "Skipped Bucket should NOT be available but was. environment.Environment: '%s', failed for '%s'", env.Name, cfg.Coordinate)
		return
	}

	if available {
		assert.Truef(t, exists, "Bucket %q should be available, but wasn't. environment.Environment: '%s', failed for '%s'", expectedId, env.Name, cfg.Coordinate)
	} else {
		assert.Falsef(t, exists, "Bucket %q should NOT be available, but was. environment.Environment: '%s', failed for '%s'", expectedId, env.Name, cfg.Coordinate)
	}

	return expectedId
}

// waitForBucketToExist tries to get bucket status until it is successfully retrieved, max tries is exhausted, or some other error occurs.
func waitForBucketToExist(ctx context.Context, client client.BucketClient, bucketName string, maxTries int) error {
	var err error
	for try := 0; try < maxTries; try++ {
		_, err = client.Get(ctx, bucketName)
		if err == nil {
			return nil
		}

		if !coreapi.IsNotFoundError(err) {
			return err
		}

		time.Sleep(time.Second)
	}
	return err
}

func wait(description string, maxPollCount int, condition func() bool) error {

	for i := 0; i <= maxPollCount; i++ {

		if condition() {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	log.Error("Error: Waiting for '%s' timed out!", description)

	return errors.New("Waiting for '" + description + "' timed out!")
}

func extractConfigName(properties parameter.Properties) (string, error) {
	val, found := properties[config.NameParameter]

	if !found {
		return "", nil
	}

	name, success := val.(string)

	if !success {
		return "", fmt.Errorf("`name` in config is not of type string")
	}

	return name, nil
}
