//go:build integration || integration_v1 || cleanup || download_restore || nightly

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

import (
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/bucket"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/resolve"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/rest"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
	"github.com/spf13/afero"
)

type entityLookup map[coordinate.Coordinate]config.ResolvedEntity

func (e entityLookup) GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool) {
	if ent, f := e.GetResolvedEntity(coordinate); f {
		if prop, f := ent.Properties[propertyName]; f {
			return prop, true
		}
	}

	return nil, false
}

func (e entityLookup) GetResolvedEntity(config coordinate.Coordinate) (config.ResolvedEntity, bool) {
	ent, f := e[config]
	return ent, f
}

// AssertAllConfigsAvailability checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificProjects []string, specificEnvironment string, available bool) {
	loadedManifest := LoadManifest(t, fs, manifestPath, specificEnvironment)

	projects := LoadProjects(t, fs, manifestPath, loadedManifest)

	envNames := make([]string, 0, len(loadedManifest.Environments))

	for _, env := range loadedManifest.Environments {
		envNames = append(envNames, env.Name)
	}

	sortedConfigs, errs := sort.ConfigsPerEnvironment(projects, envNames)
	testutils.FailTestOnAnyError(t, errs, "sorting configurations failed")

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
		for _, p := range projects {
			projectsToValidate[p.Id] = struct{}{}
		}
	}

	for envName, configs := range sortedConfigs {

		env := loadedManifest.Environments[envName]

		clients := CreateDynatraceClients(t, env)

		entities := entityLookup{}
		var parameters []parameter.NamedParameter

		for _, theConfig := range configs {
			coord := theConfig.Coordinate

			ctx := context.WithValue(context.TODO(), log.CtxKeyCoord{}, coord)
			ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: theConfig.Environment, Group: theConfig.Group})

			if theConfig.Skip {
				entities[coord] = config.ResolvedEntity{
					EntityName: coord.ConfigId,
					Coordinate: coord,
					Properties: parameter.Properties{},
					Skip:       true,
				}
				continue
			}

			configParameters, errs := sort.Parameters(theConfig.Group, theConfig.Environment, theConfig.Coordinate, theConfig.Parameters)
			testutils.FailTestOnAnyError(t, errs, "sorting of parameter values failed")

			parameters = append(parameters, configParameters...)

			properties, errs := resolve.ParameterValues(&theConfig, entities, parameters)
			testutils.FailTestOnAnyError(t, errs, "resolving of parameter values failed")

			properties[config.IdParameter] = "NO REAL ID NEEDED FOR CHECKING AVAILABILITY"

			configName, err := extractConfigName(properties)
			assert.NoError(t, err)

			entities[coord] = config.ResolvedEntity{
				EntityName: configName,
				Coordinate: coord,
				Properties: properties,
				Skip:       false,
			}

			apis := api.NewAPIs()
			if _, found := projectsToValidate[coord.Project]; found {
				switch typ := theConfig.Type.(type) {
				case config.SettingsType:
					AssertSetting(t, ctx, clients.Settings(), typ, env.Name, available, theConfig)
				case config.ClassicApiType:
					assert.NotEmpty(t, configName, "classic API config %v is missing name, can not assert if it exists", theConfig.Coordinate)
					AssertConfig(t, ctx, clients.Classic(), apis[typ.Api], env, available, theConfig, configName)
				case config.AutomationType:
					if clients.Automation() == nil {
						t.Errorf("can not assert existience of Automtation config %q (%s) because no AutomationClient exists - was the test env not configured as Platform?", theConfig.Coordinate, typ.Resource)
						return
					}
					AssertAutomation(t, *clients.Automation(), env, available, typ.Resource, theConfig)
				case config.BucketType:
					if clients.Bucket() == nil {
						t.Errorf("can not assert existience of Bucket config %q) because no BucketClient exists - was the test env not configured as Platform?", theConfig.Coordinate)
						return
					}
					AssertBucket(t, *clients.Bucket(), env, available, theConfig)
				default:
					t.Errorf("Can not assert config of unknown type %q", theConfig.Coordinate.Type)
				}
			}
		}
	}
}

func AssertConfig(t *testing.T, ctx context.Context, client dtclient.ConfigClient, theApi api.API, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config, name string) {

	configType := config.Coordinate.Type

	var exists bool

	if config.Skip {
		exists, _, _ = client.ConfigExistsByName(ctx, theApi, name)
		assert.False(t, exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := wait(description, 120, func() bool {
		exists, _, _ = client.ConfigExistsByName(ctx, theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NoError(t, err)

	if shouldBeAvailable {
		assert.True(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	} else {
		assert.False(t, exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	}
}

func AssertSetting(t *testing.T, ctx context.Context, c dtclient.SettingsClient, typ config.SettingsType, environmentName string, shouldBeAvailable bool, config config.Config) {
	expectedExtId, err := idutils.GenerateExternalID(config.Coordinate)
	if err != nil {
		t.Errorf("Unable to generate external id: %v", err)
		return
	}

	objects, err := c.ListSettings(ctx, typ.SchemaId, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == expectedExtId }})
	assert.NoError(t, err)

	if len(objects) > 1 {
		t.Errorf("Expected a specific Settings Object with externalId %q, but %d are present instead.", expectedExtId, len(objects))
		return
	}

	exists := len(objects) == 1

	if config.Skip {
		assert.False(t, exists, "Skipped Settings Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", environmentName, config.Coordinate, typ.SchemaId)
		return
	}

	if shouldBeAvailable {
		assert.True(t, exists, "Settings Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environmentName, config.Coordinate, typ.SchemaId)
	} else {
		assert.False(t, exists, "Settings Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environmentName, config.Coordinate, typ.SchemaId)
	}
}

func AssertAutomation(t *testing.T, c automation.Client, env manifest.EnvironmentDefinition, shouldBeAvailable bool, resource config.AutomationResource, cfg config.Config) {
	resourceType, err := automationutils.ClientResourceTypeFromConfigType(resource)
	assert.NoError(t, err, "failed to get resource type for: %s", cfg.Coordinate)

	var expectedId string
	if cfg.OriginObjectId != "" {
		expectedId = cfg.OriginObjectId
	} else {
		expectedId = idutils.GenerateUUIDFromCoordinate(cfg.Coordinate)
	}

	resp, err := c.List(context.TODO(), resourceType)
	assert.NoError(t, err)

	var exists bool
	for _, r := range resp {
		if r.ID == expectedId {
			exists = true
			break
		}
	}

	if cfg.Skip {
		assert.False(t, exists, "Skipped Automation Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
		return
	}

	if shouldBeAvailable {
		assert.True(t, exists, "Automation Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
	} else {
		assert.False(t, exists, "Automation Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
	}
}

func AssertBucket(t *testing.T, client bucket.Client, env manifest.EnvironmentDefinition, available bool, cfg config.Config) {

	var expectedId string
	if cfg.OriginObjectId != "" {
		expectedId = cfg.OriginObjectId
	} else {
		expectedId = fmt.Sprintf("%s_%s_%s", cfg.Coordinate.Project, cfg.Coordinate.Type, cfg.Coordinate.ConfigId)
	}

	_, err := client.Get(context.TODO(), expectedId)
	exists := true

	var respErr rest.RespError
	if errors.As(err, &respErr) {
		if respErr.StatusCode == 404 {
			exists = false
		} else {
			assert.NoError(t, err)
		}
	} else if err != nil {
		assert.NoError(t, err)
	}

	if cfg.Skip {
		assert.Falsef(t, exists, "Skipped Automation Object should NOT be available but was. environment.Environment: '%s', failed for '%s'", env.Name, cfg.Coordinate)
		return
	}

	if available {
		assert.Truef(t, exists, "Automation Object should be available, but wasn't. environment.Environment: '%s', failed for '%s'", env.Name, cfg.Coordinate)
	} else {
		assert.Falsef(t, exists, "Automation Object should NOT be available, but was. environment.Environment: '%s', failed for '%s'", env.Name, cfg.Coordinate)
	}
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
