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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/test"
	"github.com/spf13/afero"
	"regexp"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"gotest.tools/assert"
)

func TestDoCleanup(t *testing.T) {

	manifestPath := "test-resources/test_environments_manifest.yaml"

	fs := afero.NewOsFs()
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})
	test.FailTestOnAnyError(t, errs, "failed to load manifest to delete for")

	apis := api.NewApis()

	// match anything ending in test suffixes of {timestamp}_{random numbers}_{some suffix test}
	testSuffixRegex := regexp.MustCompile(`^.+_\d+_\d+_.*$`)

	for _, environment := range loadedManifest.GetEnvironmentsAsSlice() {
		deletedConfigs := 0
		token, err := environment.GetToken()
		assert.NilError(t, err)

		envUrl, err := environment.GetUrl()
		assert.NilError(t, err)

		client, err := rest.NewDynatraceClient(envUrl, token)
		assert.NilError(t, err)

		for _, api := range apis {
			if api.GetId() == "calculated-metrics-log" {
				t.Logf("Skipping cleanup of legacy log monitoring API")
				continue
			}

			values, err := client.List(api)
			assert.NilError(t, err)

			for _, value := range values {
				if testSuffixRegex.MatchString(value.Name) || testSuffixRegex.MatchString(value.Id) {
					err := client.DeleteById(api, value.Id)
					if err != nil {
						t.Errorf("failed to delete %s (%s): %v", value.Name, api.GetId(), err)
					} else {
						log.Info("Deleted %s (%s)", value.Name, api.GetId())
						deletedConfigs++
					}
				}
			}
		}
		t.Logf("Deleted %d leftover test configurations from %s (%s)", deletedConfigs, environment.Name, envUrl)
	}

}
