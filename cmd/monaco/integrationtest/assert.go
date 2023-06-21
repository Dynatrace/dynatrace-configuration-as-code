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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"testing"
	"time"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/deploy"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/topologysort"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

// AssertAllConfigsAvailability checks all configurations of a given project with given availability
func AssertAllConfigsAvailability(t *testing.T, fs afero.Fs, manifestPath string, specificProjects []string, specificEnvironment string, available bool) {
	loadedManifest := LoadManifest(t, fs, manifestPath, specificEnvironment)

	projects := LoadProjects(t, fs, manifestPath, loadedManifest)

	envNames := make([]string, 0, len(loadedManifest.Environments))

	for _, env := range loadedManifest.Environments {
		envNames = append(envNames, env.Name)
	}

	sortedConfigs, errs := topologysort.GetSortedConfigsForEnvironments(projects, envNames)
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

		entities := make(map[coordinate.Coordinate]parameter.ResolvedEntity)
		var parameters []topologysort.ParameterWithName

		for _, theConfig := range configs {
			coord := theConfig.Coordinate

			ctx := context.WithValue(context.TODO(), log.CtxKeyCoord{}, coord)
			ctx = context.WithValue(ctx, log.CtxKeyEnv{}, log.CtxValEnv{Name: theConfig.Environment, Group: theConfig.Group})

			if theConfig.Skip {
				entities[coord] = parameter.ResolvedEntity{
					EntityName: coord.ConfigId,
					Coordinate: coord,
					Properties: parameter.Properties{},
					Skip:       true,
				}
				continue
			}

			configParameters, errs := topologysort.SortParameters(theConfig.Group, theConfig.Environment, theConfig.Coordinate, theConfig.Parameters)
			testutils.FailTestOnAnyError(t, errs, "sorting of parameter values failed")

			parameters = append(parameters, configParameters...)

			properties, errs := deploy.ResolveParameterValues(&theConfig, entities, parameters)
			testutils.FailTestOnAnyError(t, errs, "resolving of parameter values failed")

			properties[config.IdParameter] = "NO REAL ID NEEDED FOR CHECKING AVAILABILITY"

			configName, err := extractConfigName(properties)
			assert.NilError(t, err)

			entities[coord] = parameter.ResolvedEntity{
				EntityName: configName,
				Coordinate: coord,
				Properties: properties,
				Skip:       false,
			}

			apis := api.NewAPIs()
			if _, found := projectsToValidate[coord.Project]; found {
				switch typ := theConfig.Type.(type) {
				case config.SettingsType:
					assertSetting(t, ctx, clients.Settings(), typ, env, available, theConfig)
				case config.ClassicApiType:
					assertConfig(t, ctx, clients.Classic(), apis[typ.Api], env, available, theConfig, configName)
				case config.AutomationType:
					if clients.Automation() == nil {
						t.Errorf("can not assert existience of Automtation config %q (%s) because no AutomationClient exists - was the test env not configured as Platform?", theConfig.Coordinate, typ.Resource)
						return
					}
					assertAutomation(t, *clients.Automation(), env, available, typ.Resource, theConfig)
				default:
					t.Errorf("Can not assert config of unknown type %q", theConfig.Coordinate.Type)
				}
			}
		}
	}
}

func assertConfig(t *testing.T, ctx context.Context, client dtclient.ConfigClient, theApi api.API, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config, name string) {

	configType := config.Coordinate.Type

	var exists bool

	if config.Skip {
		exists, _, _ = client.ConfigExistsByName(ctx, theApi, name)
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
		return
	}

	description := fmt.Sprintf("%s %s on environment %s", configType, name, environment.Name)

	// To deal with delays of configs becoming available try for max 120 polling cycles (4min - at 2sec cycles) for expected state to be reached
	err := wait(description, 120, func() bool {
		exists, _, _ = client.ConfigExistsByName(ctx, theApi, name)
		return (shouldBeAvailable && exists) || (!shouldBeAvailable && !exists)
	})
	assert.NilError(t, err)

	if shouldBeAvailable {
		assert.Check(t, exists, "Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	} else {
		assert.Check(t, !exists, "Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, name, configType)
	}
}

func assertSetting(t *testing.T, ctx context.Context, c dtclient.SettingsClient, typ config.SettingsType, environment manifest.EnvironmentDefinition, shouldBeAvailable bool, config config.Config) {
	expectedExtId, err := idutils.GenerateExternalID(config.Coordinate)
	if err != nil {
		t.Errorf("Unable to generate external id: %v", err)
		return
	}

	objects, err := c.ListSettings(ctx, typ.SchemaId, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == expectedExtId }})
	assert.NilError(t, err)

	if len(objects) > 1 {
		t.Errorf("Expected a specific Settings Object with externalId %q, but %d are present instead.", expectedExtId, len(objects))
		return
	}

	exists := len(objects) == 1

	if config.Skip {
		assert.Check(t, !exists, "Skipped Settings Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
		return
	}

	if shouldBeAvailable {
		assert.Check(t, exists, "Settings Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
	} else {
		assert.Check(t, !exists, "Settings Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", environment.Name, config.Coordinate, typ.SchemaId)
	}
}

func assertAutomation(t *testing.T, c automation.Client, env manifest.EnvironmentDefinition, shouldBeAvailable bool, resource config.AutomationResource, cfg config.Config) {
	resourceType, err := automationutils.ClientResourceTypeFromConfigType(resource)
	assert.NilError(t, err, "failed to get resource type for: %s", cfg.Coordinate)

	var expectedId string
	if cfg.OriginObjectId != "" {
		expectedId = cfg.OriginObjectId
	} else {
		expectedId = idutils.GenerateUUIDFromCoordinate(cfg.Coordinate)
	}

	resp, err := c.List(context.TODO(), resourceType)
	assert.NilError(t, err)

	var exists bool
	for _, r := range resp {
		if r.ID == expectedId {
			exists = true
			break
		}
	}

	if cfg.Skip {
		assert.Check(t, !exists, "Skipped Automation Object should NOT be available but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
		return
	}

	if shouldBeAvailable {
		assert.Check(t, exists, "Automation Object should be available, but wasn't. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
	} else {
		assert.Check(t, !exists, "Automation Object should NOT be available, but was. environment.Environment: '%s', failed for '%s' (%s)", env.Name, cfg.Coordinate, resource)
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
		return "", fmt.Errorf("missing `name` for config")
	}

	name, success := val.(string)

	if !success {
		return "", fmt.Errorf("`name` in config is not of type string")
	}

	return name, nil
}
