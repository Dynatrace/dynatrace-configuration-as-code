//go:build integration || download_restore || nightly

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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/spf13/afero"
	"path/filepath"
	"testing"
)

// RunIntegrationWithCleanup runs an integration test and cleans up the created configs afterwards
// This is done by using InMemoryFileReader, which rewrites the names of the read configs internally. It ready all the
// configs once and holds them in memory. Any subsequent modification of a config (applying them to an environment)
// is done based on the data in memory. The re-writing of config names ensures, that they have an unique name and don't
// conflict with other configs created by other integration tests.
//
// After the test run, the unique name also helps with finding the applied configs in all the environments and calling
// the respective DELETE api.
//
// The new naming scheme of created configs is defined in a transformer function. By default, this is:
//
// <original name>_<current timestamp><defined suffix>
// e.g. my-config_1605258980000_Suffix
func RunIntegrationWithCleanup(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {

	fs := testutils.CreateTestFileSystem()
	// enable automation resources feature
	envVars := map[string]string{"MONACO_FEAT_AUTOMATION_RESOURCES": "1"}
	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, envVars, testFunc)
}

func RunIntegrationWithCleanupOnGivenFs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc func(fs afero.Fs)) {
	runIntegrationWithCleanup(t, testFs, configFolder, manifestPath, specificEnvironment, suffixTest, nil, testFunc)
}

func RunIntegrationWithCleanupGivenEnvs(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	fs := testutils.CreateTestFileSystem()

	runIntegrationWithCleanup(t, fs, configFolder, manifestPath, specificEnvironment, suffixTest, envVars, testFunc)
}

func runIntegrationWithCleanup(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc func(fs afero.Fs)) {
	var envs []string
	if len(specificEnvironment) > 0 {
		envs = append(envs, specificEnvironment)
	}

	loadedManifest, errs := manifest.LoadManifest(&manifest.LoaderContext{
		Fs:           testFs,
		ManifestPath: manifestPath,
		Environments: envs,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of manifest failed")

	configFolder, _ = filepath.Abs(configFolder)

	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, testFs, configFolder, suffixTest)

	t.Cleanup(func() {
		integrationtest.CleanupIntegrationTest(t, testFs, manifestPath, loadedManifest, suffix)
	})

	for k, v := range envVars {
		setTestEnvVar(t, k, v, suffix)
	}

	setTestEnvVar(t, "UNIQUE_TEST_SUFFIX", suffix, suffix)

	testFunc(testFs)
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	suffix := integrationtest.GenerateTestSuffix(generalSuffix)
	transformers := []func(line string) string{
		func(name string) string {
			return integrationtest.ReplaceName(name, integrationtest.AddSuffix(suffix))
		},
		func(id string) string {
			return integrationtest.ReplaceId(id, integrationtest.AddSuffix(suffix))
		},
	}

	err := integrationtest.RewriteConfigNames(configFolder, fs, transformers)
	if err != nil {
		t.Fatalf("Error rewriting configs names: %s", err)
		return suffix
	}

	return suffix
}

func setTestEnvVar(t *testing.T, key, value, testSuffix string) {
	t.Setenv(key, value)                                   // expose directly
	t.Setenv(fmt.Sprintf("%s_%s", key, testSuffix), value) // expose with suffix (env parameter "name" is subject to rewrite)
}
