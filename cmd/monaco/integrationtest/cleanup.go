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
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
)

// Deletes all configs that end with a test suffix and any Settings created by the given manifest
func CleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, suffix string) {

	log.Info("### Cleaning up after integration test ###")

	apis := api.NewAPIs()
	suffix = "_" + suffix

	for _, environment := range loadedManifest.Environments {

		c := CreateDynatraceClient(t, environment)

		cleanupSettings(t, fs, manifestPath, loadedManifest, environment.Name, c)
		cleanupConfigs(t, apis, c, suffix)
	}
}

func cleanupConfigs(t *testing.T, apis api.APIs, c client.ConfigClient, suffix string) {
	for _, api := range apis {
		if api.ID == "calculated-metrics-log" {
			t.Logf("Skipping cleanup of legacy log monitoring API")
			continue
		}

		values, err := c.ListConfigs(api)
		if err != nil {
			t.Logf("Failed to cleanup any test configs of type %q: %v", api.ID, err)
		}

		for _, value := range values {
			// For the calculated-metrics-log API, the suffix is part of the ID, not name
			if strings.HasSuffix(value.Name, suffix) || strings.HasSuffix(value.Id, suffix) {
				err := c.DeleteConfigById(api, value.Id)
				if err != nil {
					t.Logf("Failed to cleanup test config: %s (%s): %v", value.Name, api.ID, err)
				} else {
					log.Info("Cleaned up test config %s (%s)", value.Name, value.Id)
				}
			}
		}
	}
}

func cleanupSettings(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, environment string, c client.SettingsClient) {
	projects := LoadProjects(t, fs, manifestPath, loadedManifest)
	for _, p := range projects {
		cfgsForEnv, ok := p.Configs[environment]
		if !ok {
			t.Logf("Failed to cleanup Settings for env %s - no configs found", environment)
		}
		for _, configs := range cfgsForEnv {
			for _, cfg := range configs {
				if cfg.Type.IsSettings() {
					extID := idutils.GenerateExternalID(cfg.Type.SchemaId, cfg.Coordinate.ConfigId)
					deleteSettingsObjects(t, cfg.Type.SchemaId, extID, c)
				}
			}
		}
	}
}

func deleteSettingsObjects(t *testing.T, schema, externalID string, c client.SettingsClient) {
	objects, err := c.ListSettings(schema, client.ListSettingsOptions{DiscardValue: true, Filter: func(o client.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
	if err != nil {
		t.Logf("Failed to cleanup test config: could not fetch settings 2.0 objects with schema ID %s: %v", schema, err)
		return
	}

	if len(objects) == 0 {
		t.Logf("No %s settings object found to cleanup: %s", schema, externalID)
		return
	}

	for _, obj := range objects {
		err := c.DeleteSettings(obj.ObjectId)
		if err != nil {
			t.Logf("Failed to cleanup test config: could not delete settings 2.0 object with object ID %s: %v", obj.ObjectId, err)
		} else {
			log.Info("Cleaned up test Setting %s (%s)", externalID, schema)
		}
	}
}
