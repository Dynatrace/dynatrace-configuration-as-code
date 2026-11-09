//go:build integration

/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvaultv2

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/client"
	manifestloader "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest/loader"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestCredentialVaultV2(t *testing.T) {
	configFolder := "testdata/"
	manifestPath := configFolder + "manifest.yaml"
	environment := "platform_env"

	t.Run("Simple deployment creates the credential", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithManifestPath(manifestPath),
				runner.WithSuffix("CredentialVaultV2"),
				runner.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				// when deploying once
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=single-credential --verbose", manifestPath))
				assert.NoError(t, err)

				credentialVaultClient := createCredentialVaultClient(t, fs, manifestPath, environment)
				result, err := credentialVaultClient.List(t.Context())
				require.NoError(t, err)

				// credentials are identified by name, as the API does not support external IDs
				assertCredentialCount(t, 1, result, "Sample Credentials_"+testContext.Suffix)
			})
	})

	t.Run("Deploying the config twice does not create a second credential", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithManifestPath(manifestPath),
				runner.WithSuffix("CredentialVaultV2"),
				runner.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				// when deploying twice
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=single-credential --verbose", manifestPath))
				assert.NoError(t, err)

				err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=single-credential --verbose", manifestPath))
				assert.NoError(t, err)

				credentialVaultClient := createCredentialVaultClient(t, fs, manifestPath, environment)
				result, err := credentialVaultClient.List(t.Context())
				require.NoError(t, err)

				assertCredentialCount(t, 1, result, "Sample Credentials_"+testContext.Suffix)
			})
	})

	t.Run("When deploying two configs, two configs exist", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithManifestPath(manifestPath),
				runner.WithSuffix("CredentialVaultV2"),
				runner.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				// when deploying twice, just to make sure
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=two-credentials --verbose", manifestPath))
				assert.NoError(t, err)

				err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=two-credentials --verbose", manifestPath))
				assert.NoError(t, err)

				credentialVaultClient := createCredentialVaultClient(t, fs, manifestPath, environment)
				result, err := credentialVaultClient.List(t.Context())
				require.NoError(t, err)

				assertCredentialCount(t, 1, result, "My Credential_"+testContext.Suffix)
				assertCredentialCount(t, 1, result, "Second Credential_"+testContext.Suffix)
			})
	})

	t.Run("Credentials can be referenced from other configs", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithManifestPath(manifestPath),
				runner.WithSuffix("CredentialVaultV2"),
				runner.WithEnvironment(environment),
			},
			func(fs afero.Fs, testContext runner.TestContext) {
				// when deploying once
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --project=referenced-credential --verbose", manifestPath))
				assert.NoError(t, err)

				credentialVaultClient := createCredentialVaultClient(t, fs, manifestPath, environment)
				result, err := credentialVaultClient.List(t.Context())
				require.NoError(t, err)

				assertCredentialCount(t, 1, result, "Referenced Credential_"+testContext.Suffix)
			})
	})
}

func createCredentialVaultClient(t *testing.T, fs afero.Fs, manifestPath string, environment string) client.CredentialVaultClient {
	man, errs := manifestloader.Load(&manifestloader.Context{
		Fs:           fs,
		ManifestPath: manifestPath,
		Opts:         manifestloader.Options{RequireEnvironmentGroups: true},
	})
	require.Empty(t, errs)

	clientSet := runner.CreateDynatraceClients(t, man.Environments.SelectedEnvironments[environment])
	return clientSet.CredentialVaultClient
}

type credentialResponse struct {
	Name string `json:"name"`
}

func parsePayload(t *testing.T, data []byte) credentialResponse {
	var credResp credentialResponse

	err := json.Unmarshal(data, &credResp)
	require.NoError(t, err)

	return credResp
}

func extractCredentialNames(t *testing.T, response api.PagedListResponse) []string {
	allObjects := response.All()
	names := make([]string, 0, len(allObjects))
	for _, obj := range allObjects {
		names = append(names, parsePayload(t, obj).Name)
	}
	return names
}

func assertCredentialCount(t *testing.T, expected int, response api.PagedListResponse, name string) {
	count := 0
	for _, c := range extractCredentialNames(t, response) {
		if c == name {
			count++
		}
	}

	assert.Equal(t, expected, count, "unexpected number of credentials with name '%s'", name)
}
