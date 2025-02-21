//go:build integration

/*
 * @license
 * Copyright 2025 Dynatrace LLC
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
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
)

func TestSegments(t *testing.T) {

	configFolder := "test-resources/segments/"
	manifestPath := configFolder + "manifest.yaml"
	environment := "platform_env"

	// enable FF
	t.Setenv(featureflags.Segments.EnvName(), "true")

	t.Run("Simple deployment creates the segment", func(t *testing.T) {

		RunIntegrationWithCleanup(t, configFolder, manifestPath, environment, "Segments", func(fs afero.Fs, testContext TestContext) {
			// when deploying once
			err := monaco.RunWithFSf(fs, "monaco deploy %s --project=standalone-segment --verbose", manifestPath)
			assert.NoError(t, err)

			segmentsClient := createSegmentsClient(t, fs, manifestPath, environment)
			result, err := segmentsClient.GetAll(t.Context())
			assert.NoError(t, err)

			coord := coordinate.Coordinate{
				Project:  "standalone-segment",
				Type:     "segment",
				ConfigId: "my-segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, true, result, coord)
		})
	})

	t.Run("Deploying the config twice does not create a second segment", func(t *testing.T) {
		RunIntegrationWithCleanup(t, configFolder, manifestPath, environment, "Segments", func(fs afero.Fs, testContext TestContext) {
			// when deploying twice
			err := monaco.RunWithFSf(fs, "monaco deploy %s --project=standalone-segment --verbose", manifestPath)
			assert.NoError(t, err)

			err = monaco.RunWithFSf(fs, "monaco deploy %s --project=standalone-segment --verbose", manifestPath)
			assert.NoError(t, err)

			segmentsClient := createSegmentsClient(t, fs, manifestPath, environment)
			result, err := segmentsClient.GetAll(t.Context())
			assert.NoError(t, err)

			coord := coordinate.Coordinate{
				Project:  "standalone-segment",
				Type:     "segment",
				ConfigId: "my-segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, true, result, coord)
		})
	})

	t.Run("When deploying two configs, two configs exist", func(t *testing.T) {
		RunIntegrationWithCleanup(t, configFolder, manifestPath, environment, "Segments", func(fs afero.Fs, testContext TestContext) {
			// when deploying twice, just to make sure
			err := monaco.RunWithFSf(fs, "monaco deploy %s --project=two-segments --verbose", manifestPath)
			assert.NoError(t, err)

			err = monaco.RunWithFSf(fs, "monaco deploy %s --project=two-segments --verbose", manifestPath)
			assert.NoError(t, err)

			segmentsClient := createSegmentsClient(t, fs, manifestPath, environment)
			result, err := segmentsClient.GetAll(t.Context())
			assert.NoError(t, err)

			coord := coordinate.Coordinate{
				Project:  "two-segments",
				Type:     "segment",
				ConfigId: "my-segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, true, result, coord)

			coord = coordinate.Coordinate{
				Project:  "two-segments",
				Type:     "segment",
				ConfigId: "second-segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, true, result, coord)
		})
	})

	t.Run("With a disabled FF the deploy should fail", func(t *testing.T) {
		t.Setenv(featureflags.Segments.EnvName(), "false")

		RunIntegrationWithoutCleanup(t, configFolder, manifestPath, environment, "Segments", func(fs afero.Fs, testContext TestContext) {
			// when deploying once
			err := monaco.RunWithFSf(fs, "monaco deploy %s --project=standalone-segment --verbose", manifestPath)
			assert.Error(t, err)

			segmentsClient := createSegmentsClient(t, fs, manifestPath, environment)
			result, err := segmentsClient.GetAll(t.Context())
			assert.NoError(t, err)

			coord := coordinate.Coordinate{
				Project:  "standalone-segment",
				Type:     "segment",
				ConfigId: "my-segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, false, result, coord)
		})
	})

	t.Run("Segments can be referenced from other configs", func(t *testing.T) {

		RunIntegrationWithCleanup(t, configFolder, manifestPath, environment, "Segments", func(fs afero.Fs, testContext TestContext) {
			// when deploying once
			err := monaco.RunWithFSf(fs, "monaco deploy %s --project=referenced-segment --verbose", manifestPath)
			assert.NoError(t, err)

			segmentsClient := createSegmentsClient(t, fs, manifestPath, environment)
			result, err := segmentsClient.GetAll(t.Context())
			assert.NoError(t, err)

			coord := coordinate.Coordinate{
				Project:  "referenced-segment",
				Type:     "segment",
				ConfigId: "segment_" + testContext.suffix,
			}
			assertSegmentIsInResponse(t, true, result, coord)
		})
	})
}

func createSegmentsClient(t *testing.T, fs afero.Fs, manifestPath string, environment string) client.SegmentClient {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	clientSet := integrationtest.CreateDynatraceClients(t, man.Environments[environment])
	return clientSet.SegmentClient
}

type segmentsResponse struct {
	ExternalId string `json:"externalId"`
}

func parseSegmentsPayload(t *testing.T, resp api.Response) segmentsResponse {
	segResp := segmentsResponse{}

	err := json.Unmarshal(resp.Data, &segResp)
	assert.NoError(t, err)

	return segResp
}

func assertSegmentIsInResponse(t *testing.T, present bool, responses []api.Response, coord coordinate.Coordinate) {
	externalId, err := idutils.GenerateExternalID(coord)
	assert.NoError(t, err)

	found := false

	for _, resp := range responses {
		payload := parseSegmentsPayload(t, resp)

		if payload.ExternalId == externalId {
			found = true
			break
		}
	}

	if found == present {
		return
	}

	if !found {
		assert.Fail(t, "Segment not present", "Segment with externalID '%s' not present in response", externalId)
	}

	assert.Fail(t, "Segment present", "Segment with externalID '%s' is present in response but should not be", externalId)
}
