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
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
)

type TestContext struct {
	suffix string
}

type testOptions struct {
	fs                                                      afero.Fs
	configFolder, manifestPath, specificEnvironment, suffix string
	envVars                                                 map[string]string

	// skipCleanup skips the Monaco cleanup that generates the delete file and deletes all created resources.
	// It is false by default, thus the cleanup must be disabled intentionally
	skipCleanup bool
}

type TestFunc func(fs afero.Fs, ctx TestContext)

type Option func(options *testOptions)
type Options []Option

func WithFs(fs afero.Fs) Option {
	return func(options *testOptions) {
		options.fs = fs
	}
}

func WithManifest(relativePath string) Option {
	return func(options *testOptions) {
		options.manifestPath = path.Join(options.configFolder, relativePath)
	}
}

func WithoutCleanup() Option {
	return func(options *testOptions) {
		options.skipCleanup = true
	}
}
func WithSuffix(suffix string) Option {
	return func(options *testOptions) {
		options.suffix = suffix
	}
}

func WithEnvironment(env string) Option {
	return func(options *testOptions) {
		options.specificEnvironment = env
	}
}

func WithEnvVars(env map[string]string) Option {
	return func(options *testOptions) {
		for k, v := range env {
			options.envVars[k] = v
		}
	}
}

func WithEnvVar(k, v string) Option {
	return func(options *testOptions) {
		options.envVars[k] = v
	}
}

// Run is the main entry point for integration tests.
//
// Run uses the [workingDirectory] as the main working directory to execute the test-cases.
// Run rewrites the names and IDs of all configurations to include a randomly-generated suffix. This ensures that they have a unique name and do not confliect with each other config when running integration tests in parallel.
// Run then executes the provided TestFunc which should contain the integration test.
// After the test ran, the configurations are automatically deleted.
//
// By default, Run
//   - utilizes an in-memory file system, which copies the files read into an in-memory data structure. All subsequent writes will be written to this in-memory data structure. This behaviro can be overwritten using [WithFs].
//   - uses the file `manifest.yaml` as default manifest file inside the working-directory. This can be overwritten using [WithManifest]
//   - cleans up configurations. This can be overwritten using [WithoutCleanup].
//   - uses the suffix in the form of `<original name>_<current timestamp><defined suffix>`. The <defined suffix> can be overwritten using [WithSuffix].
//   - does not provide additional environment variables. You can use [WithEnvVars], [WithEnvVar], [WithMonitoredEntity].
//   - uses all environments for deletion. This can be overwritten using [WithEnvironment]
func Run(t *testing.T, workingDirectory string, opts Options, fn TestFunc) {
	options := testOptions{
		fs:                  testutils.CreateTestFileSystem(),
		configFolder:        workingDirectory,
		manifestPath:        path.Join(workingDirectory, "manifest.yaml"),
		specificEnvironment: "",
		suffix:              t.Name(),
		envVars:             nil,
	}

	for _, optFn := range opts {
		optFn(&options)
	}

	runIntegration(t, options, fn)
}

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
//
// Deprecated: Use Run instead with
//
//	Run(t, configFolder, Options{WithEnvironment, WithManifest, WithSuffix}, testFunc)
func RunIntegrationWithCleanup(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc TestFunc) {
	opts := testOptions{
		fs:                  testutils.CreateTestFileSystem(),
		configFolder:        configFolder,
		manifestPath:        manifestPath,
		specificEnvironment: specificEnvironment,
		suffix:              suffixTest,
		envVars:             nil,
	}

	runIntegration(t, opts, testFunc)
}

func RunIntegrationWithoutCleanup(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc TestFunc) {
	opts := testOptions{
		fs:                  testutils.CreateTestFileSystem(),
		configFolder:        configFolder,
		manifestPath:        manifestPath,
		specificEnvironment: specificEnvironment,
		suffix:              suffixTest,
		envVars:             nil,
		skipCleanup:         true,
	}

	runIntegration(t, opts, testFunc)
}

func RunIntegrationWithCleanupOnGivenFs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, testFunc TestFunc) {
	opts := testOptions{
		fs:                  testFs,
		configFolder:        configFolder,
		manifestPath:        manifestPath,
		specificEnvironment: specificEnvironment,
		suffix:              suffixTest,
		envVars:             nil,
	}

	runIntegration(t, opts, testFunc)
}

func RunIntegrationWithCleanupOnGivenFsAndEnvs(t *testing.T, testFs afero.Fs, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc TestFunc) {
	opts := testOptions{
		fs:                  testFs,
		configFolder:        configFolder,
		manifestPath:        manifestPath,
		specificEnvironment: specificEnvironment,
		suffix:              suffixTest,
		envVars:             envVars,
	}

	runIntegration(t, opts, testFunc)
}

func RunIntegrationWithCleanupGivenEnvs(t *testing.T, configFolder, manifestPath, specificEnvironment, suffixTest string, envVars map[string]string, testFunc TestFunc) {
	opts := testOptions{
		fs:                  testutils.CreateTestFileSystem(),
		configFolder:        configFolder,
		manifestPath:        manifestPath,
		specificEnvironment: specificEnvironment,
		suffix:              suffixTest,
		envVars:             envVars,
	}

	runIntegration(t, opts, testFunc)

}

func runIntegration(t *testing.T, opts testOptions, testFunc TestFunc) {
	configFolder, _ := filepath.Abs(opts.configFolder)

	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, opts.fs, configFolder, opts.suffix)

	for k, v := range opts.envVars {
		setTestEnvVar(t, k, v, suffix)
	}

	if !opts.skipCleanup {
		defer func() {
			integrationtest.CleanupIntegrationTest(t, opts.fs, opts.manifestPath, opts.specificEnvironment, suffix)
		}()
	}

	setTestEnvVar(t, "UNIQUE_TEST_SUFFIX", suffix, suffix)

	testFunc(opts.fs, TestContext{
		suffix: suffix,
	})
}

func appendUniqueSuffixToIntegrationTestConfigs(t *testing.T, fs afero.Fs, configFolder string, generalSuffix string) string {
	suffix := integrationtest.GenerateTestSuffix(t, generalSuffix)
	transformers := []func(line string) string{
		func(name string) string {
			return integrationtest.ReplaceName(name, integrationtest.GetAddSuffixFunction(suffix))
		},
		func(id string) string {
			return integrationtest.ReplaceId(id, integrationtest.GetAddSuffixFunction(suffix))
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
	t.Setenv(key, value)                                        // expose directly
	t.Setenv(integrationtest.AddSuffix(key, testSuffix), value) // expose with suffix (env parameter "name" is subject to rewrite)
}

func isHardeningEnvironment() bool {
	env := os.Getenv("TEST_ENVIRONMENT")

	return env == "hardening"
}
