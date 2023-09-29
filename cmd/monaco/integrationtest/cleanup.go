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
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code-core/api/clients/buckets"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/automationutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/slices"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client/dtclient"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2/sort"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	"github.com/spf13/afero"
)

// CleanupIntegrationTest deletes all configs that end with a test suffix and any Settings created by the given manifest
func CleanupIntegrationTest(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, suffix string) {
	log.Info("### Cleaning up after integration test ###")

	apis := api.NewAPIs()
	suffix = "_" + suffix

	for _, environment := range loadedManifest.Environments {

		clients := CreateDynatraceClients(t, environment)

		cleanupByGeneratedID(t, fs, manifestPath, loadedManifest, environment.Name, clients)
		cleanupByNameSuffix(t, apis, clients.Classic(), suffix)
	}
}

// cleanupByNameSuffix removes Classic Config API test configurations if their name ends with the defined suffix
func cleanupByNameSuffix(t *testing.T, apis api.APIs, c dtclient.ConfigClient, suffix string) {
	for _, api := range apis {
		if api.ID == "calculated-metrics-log" {
			t.Logf("Skipping cleanup of legacy log monitoring API")
			continue
		}

		values, err := c.ListConfigs(context.TODO(), api)
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

// cleanupByGeneratedID removes test configurations of a given manifest's projects by their generated identifiers
func cleanupByGeneratedID(t *testing.T, fs afero.Fs, manifestPath string, loadedManifest manifest.Manifest, environment string, clients *client.ClientSet) {
	projects := LoadProjects(t, fs, manifestPath, loadedManifest)
	cfgs, errs := sort.ConfigsPerEnvironment(projects, []string{environment}) // delete in order if things have dependencies
	assert.Empty(t, errs)

	configs, ok := cfgs[environment]
	if !ok {
		t.Logf("Failed to cleanup Settings for env %s - no configs found", environment)
	}

	//ensure dependent configs are removed in the right (reverse) order - if A depends on B, sorted configs contain B first, we want to cleanly delete A before B though
	configs = slices.Reverse(configs)

	for _, cfg := range configs {
		if cfg.Skip {
			continue // if config was not deployed to this env, no need to clean up
		}
		switch typ := cfg.Type.(type) {
		case config.SettingsType:
			if cfg.OriginObjectId != "" {
				deleteSettingsObjects(t, typ.SchemaId, cfg.OriginObjectId, clients.Settings())
				continue
			}

			extID, err := idutils.GenerateExternalID(cfg.Coordinate)
			if err != nil {
				t.Log(err)
				continue
			}
			deleteSettingsObjects(t, typ.SchemaId, extID, clients.Settings())
		case config.AutomationType:
			if cfg.OriginObjectId != "" {
				deleteAutomation(t, typ.Resource, cfg.OriginObjectId, clients.Automation())
				continue
			}

			id := idutils.GenerateUUIDFromCoordinate(cfg.Coordinate)
			deleteAutomation(t, typ.Resource, id, clients.Automation())
		case config.BucketType:
			if cfg.OriginObjectId != "" {
				deleteBucket(t, cfg.OriginObjectId, clients.Bucket())
				continue
			}

			id := idutils.GenerateBucketName(cfg.Coordinate)
			deleteBucket(t, id, clients.Bucket())
		}
	}

}

func deleteSettingsObjects(t *testing.T, schema, externalID string, c dtclient.SettingsClient) {
	objects, err := c.ListSettings(context.TODO(), schema, dtclient.ListSettingsOptions{DiscardValue: true, Filter: func(o dtclient.DownloadSettingsObject) bool { return o.ExternalId == externalID }})
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

func deleteAutomation(t *testing.T, resource config.AutomationResource, id string, c *automation.Client) {
	if c == nil {
		t.Logf("not cleaning up automation %s:%s - no client defined", resource, id)
		return
	}

	resourceType, err := automationutils.ClientResourceTypeFromConfigType(resource)
	if err != nil {
		t.Logf("Unable to delete Automation config %s (%s): %v", id, resource, err)
		return
	}
	resp, err := c.Delete(context.Background(), resourceType, id)
	if err != nil {
		t.Logf("Failed to cleanup test config: could not delete Automation (%s) object with ID %s: %v", resource, id, err)
		return
	}
	if err, isAPIErr := resp.AsAPIError(); isAPIErr {
		t.Logf("Failed to cleanup test config: could not delete Automation (%s) object with ID %s: %v", resource, id, err)
		return
	}

	log.Info("Cleaned up test Automation %s (%s)", id, resource)
}

func deleteBucket(t *testing.T, bucketName string, c *buckets.Client) {
	if c == nil {
		t.Logf("not cleaning up bucket %s - no client defined", bucketName)
		return
	}

	r, err := c.Delete(context.Background(), bucketName)

	if err != nil {
		t.Logf("Unable to delete Bucket with name %s: %v", bucketName, err)
		return
	}
	if apiErr, isErr := r.AsAPIError(); isErr {
		t.Logf("Unable to delete Bucket with name %s: %v", bucketName, apiErr)
	} else {
		log.Info("Cleaned up test Bucket %s", bucketName)
	}
}
