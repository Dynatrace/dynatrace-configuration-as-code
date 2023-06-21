//go:build cleanup

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

package v2

import (
	"context"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
	"regexp"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"gotest.tools/assert"
)

func TestDoCleanup(t *testing.T) {

	manifestPath := "test-resources/test_environments_manifest.yaml"

	fs := afero.NewOsFs()
	loadedManifest, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	testutils.FailTestOnAnyError(t, errs, "failed to load manifest to delete for")

	apis := api.NewAPIs()

	// match anything ending in test suffixes of {timestamp}_{random numbers}_{some suffix test}
	testSuffixRegex := regexp.MustCompile(`^.+_\d+_\d+_.*$`)

	for _, environment := range loadedManifest.Environments {
		envUrl := environment.URL.Value

		clients := integrationtest.CreateDynatraceClients(t, environment)

		deletedConfigs := cleanupTestConfigs(t, apis, clients.Classic(), testSuffixRegex)
		t.Logf("Deleted %d leftover test configurations from %s (%s)", deletedConfigs, environment.Name, envUrl)

		deletedSettings := cleanupTestSettings(t, clients.Settings())
		t.Logf("Deleted %d leftover test Settings from %s (%s)", deletedSettings, environment.Name, envUrl)
	}

}

func cleanupTestConfigs(t *testing.T, apis api.APIs, client dtclient.ConfigClient, testSuffixRegex *regexp.Regexp) int {
	deletedConfigs := 0
	for _, api := range apis {
		if api.ID == "calculated-metrics-log" {
			t.Logf("Skipping cleanup of legacy log monitoring API")
			continue
		}

		values, err := client.ListConfigs(context.TODO(), api)
		assert.NilError(t, err)

		for _, value := range values {
			if testSuffixRegex.MatchString(value.Name) || testSuffixRegex.MatchString(value.Id) {
				err := client.DeleteConfigById(api, value.Id)
				if err != nil {
					t.Errorf("failed to delete %s (%s): %v", value.Name, api.ID, err)
				} else {
					log.Info("Deleted %s (%s)", value.Name, api.ID)
					deletedConfigs++
				}
			}
		}
	}
	return deletedConfigs
}

func cleanupTestSettings(t *testing.T, c dtclient.SettingsClient) int {
	deletedSettings := 0

	schemas, err := c.ListSchemas()
	assert.NilError(t, err)

	for _, s := range schemas {
		schemaId := s.SchemaId
		objects, err := c.ListSettings(context.TODO(), schemaId, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId != "" }})
		if err != nil {
			t.Errorf("could not fetch settings 2.0 objects with schema %s: %v", schemaId, err)
		}

		if len(objects) == 0 {
			continue
		}

		for _, obj := range objects {
			err := c.DeleteSettings(obj.ObjectId)
			if err != nil {
				t.Errorf("failed to delete %q object: %s (extId: %s): %v", obj.ObjectId, obj.ExternalId, schemaId, err)
			} else {
				log.Info("Deleted %q object: %s (extId: %s)", obj.ObjectId, obj.ExternalId, schemaId)
				deletedSettings++
			}
		}
	}

	return deletedSettings

}
