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

package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

func TestDryRunWithOAuth(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("AllConfigs"),
			runner.WithEnvironment("platform_oauth_env"),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			dryRun(t, fs, manifest, "platform_oauth_env")
		})
}

func TestDryRunWithPlatformToken(t *testing.T) {
	configFolder := "testdata/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	runner.Run(t, configFolder,
		runner.Options{
			runner.WithManifestPath(manifest),
			runner.WithSuffix("AllConfigs"),
			runner.WithEnvironment("platform_token_env"),
			runner.WithEnvVars(map[string]string{
				featureflags.PlatformToken.EnvName(): "true",
			}),
		},
		func(fs afero.Fs, _ runner.TestContext) {
			dryRun(t, fs, manifest, "platform_token_env")
		})
}

func TestDryRunWithEnvRequirement(t *testing.T) {
	configFolder := "testdata/env_requirements/"
	manifest := configFolder + "manifest.yaml"

	t.Run("only environmentGroup env vars are validated", func(t *testing.T) {
		runner.Run(t, configFolder,
			runner.Options{
				runner.WithManifestPath(manifest),
				runner.WithSuffix("ENV_REQUIREMENTS_ENV_GROUP"),
				runner.WithEnvVars(map[string]string{
					"ENVIRONMENT_SECRET": "secret",
				}),
			},
			func(fs afero.Fs, _ runner.TestContext) {
				dryRun(t, fs, manifest, "")
			})
	})

	t.Run("only account env vars are validated", func(t *testing.T) {
		t.Setenv("ACCOUNT_SECRET", "11111111-1111-1111-1111-111111111111") // valid uuid
		err := monaco.Run(t, afero.NewOsFs(), fmt.Sprintf("monaco account deploy -m %s --verbose --dry-run", manifest))
		assert.NoError(t, err)
	})
}

func dryRun(t *testing.T, fs afero.Fs, manifest string, environment string) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		t.Fatalf("unexpected HTTP request made during dry run: %s", req.RequestURI)
	}))
	defer server.Close()

	// ensure all URLs used in the manifest point at the test server
	setAllURLEnvironmentVariables(t, server.URL)

	// This causes a POST for all configs:
	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose --dry-run", manifest, environment))
	assert.NoError(t, err)

	// This causes a PUT for all configs:
	err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose --dry-run", manifest, environment))
	assert.NoError(t, err)
}

func setAllURLEnvironmentVariables(t *testing.T, url string) {
	t.Setenv("URL_ENVIRONMENT_1", url)
	t.Setenv("URL_ENVIRONMENT_2", url)
	t.Setenv("PLATFORM_URL_ENVIRONMENT_1", url)
	t.Setenv("PLATFORM_URL_ENVIRONMENT_2", url)
	t.Setenv("OAUTH_TOKEN_ENDPOINT", url)
}
