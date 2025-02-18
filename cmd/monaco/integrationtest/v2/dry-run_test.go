//go:build integration

/*
 * @license
 * Copyright 2024 Dynatrace LLC
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
)

func TestDryRun(t *testing.T) {
	specificEnvironment := "platform_env"
	configFolder := "test-resources/integration-all-configs/"
	manifest := configFolder + "manifest.yaml"

	envVars := map[string]string{
		featureflags.OpenPipeline.EnvName(): "true",
	}

	RunIntegrationWithCleanupGivenEnvs(t, configFolder, manifest, specificEnvironment, "AllConfigs", envVars, func(fs afero.Fs, _ TestContext) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			t.Fatalf("unexpected HTTP request made during dry run: %s", req.RequestURI)
		}))
		defer server.Close()

		// ensure all URLs used in the manifest point at the test server
		setAllURLEnvironmentVariables(t, server.URL)

		// This causes a POST for all configs:
		err := monaco.RunWithFs(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose --dry-run", manifest, specificEnvironment))
		assert.NoError(t, err)

		// This causes a PUT for all configs:
		err = monaco.RunWithFs(t, fs, fmt.Sprintf("monaco deploy %s --environment=%s --verbose --dry-run", manifest, specificEnvironment))
		assert.NoError(t, err)
	})
}

func setAllURLEnvironmentVariables(t *testing.T, url string) {
	t.Setenv("URL_ENVIRONMENT_1", url)
	t.Setenv("URL_ENVIRONMENT_2", url)
	t.Setenv("PLATFORM_URL_ENVIRONMENT_1", url)
	t.Setenv("PLATFORM_URL_ENVIRONMENT_2", url)
	t.Setenv("OAUTH_TOKEN_ENDPOINT", url)
}
