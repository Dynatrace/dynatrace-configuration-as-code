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

package slov2

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/idutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestSloV2(t *testing.T) {
	configFolder := "testdata/v2/"
	manifestPath := configFolder + "manifest.yaml"
	environment := "platform_env"
	project := "project"

	// enable FF
	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")

	t.Run("When deploying two configs, two configs exist", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithSuffix("SLO-V2"),
				runner.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=%s --verbose", manifestPath, project))
				assert.NoError(t, err)

				err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=%s --verbose", manifestPath, project))
				assert.NoError(t, err)

				sloV2Client := createSloV2Client(t, fs, manifestPath, environment)
				result, err := sloV2Client.List(context.TODO())
				assert.NoError(t, err)
				externalIDs := extractExternalIDs(t, result)

				cSliCoord := coordinate.Coordinate{
					Project:  project,
					Type:     string(config.ServiceLevelObjectiveID),
					ConfigId: "custom-sli_" + testContext.Suffix,
				}
				sliRefCoord := coordinate.Coordinate{
					Project:  project,
					Type:     string(config.ServiceLevelObjectiveID),
					ConfigId: "sli-reference_" + testContext.Suffix,
				}

				cSliExternalID := idutils.GenerateExternalID(cSliCoord)
				sliRefExternalID := idutils.GenerateExternalID(sliRefCoord)

				assert.Contains(t, externalIDs, cSliExternalID)
				assert.Contains(t, externalIDs, sliRefExternalID)
			})
	})

	t.Run("With a disabled FF the deploy should fail", func(t *testing.T) {
		t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "false")

		runner.Run(t, configFolder,
			runner.Options{
				runner.WithEnvironment(environment),
				runner.WithoutCleanup(),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				// when deploying once
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=%s --verbose", manifestPath, project))
				assert.Error(t, err)

				sloV2Client := createSloV2Client(t, fs, manifestPath, environment)
				result, err := sloV2Client.List(context.TODO())
				assert.NoError(t, err)
				externalIDs := extractExternalIDs(t, result)

				coord := coordinate.Coordinate{
					Project:  project,
					Type:     string(config.ServiceLevelObjectiveID),
					ConfigId: "custom-sli_" + testContext.Suffix,
				}
				externalID := idutils.GenerateExternalID(coord)
				assert.NotContains(t, externalIDs, externalID)
			})
	})
}

func extractExternalIDs(t *testing.T, response api.PagedListResponse) []string {
	allObjects := response.All()
	externalIds := make([]string, 0, len(allObjects))
	for _, obj := range allObjects {
		externalIds = append(externalIds, extractSloV2ExternalId(t, obj))
	}
	return externalIds
}

func createSloV2Client(t *testing.T, fs afero.Fs, manifestPath string, environment string) client.ServiceLevelObjectiveClient {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	assert.Empty(t, errs)

	clientSet := runner.CreateDynatraceClients(t, man.Environments.SelectedEnvironments[environment])
	return clientSet.ServiceLevelObjectiveClient
}

type sloV2Response struct {
	ExternalId string `json:"externalId"`
}

func extractSloV2ExternalId(t *testing.T, data []byte) string {
	sloResp := sloV2Response{}

	err := json.Unmarshal(data, &sloResp)
	assert.NoError(t, err)

	return sloResp.ExternalId
}
