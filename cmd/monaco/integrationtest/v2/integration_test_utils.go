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

// TestContext contains all information necessary for the test-run
type TestContext struct {
	// suffix contains the suffix which is generated for the test-run.
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

// TestFunc is the function which is executed for the test-run.
// Within this function, all Monaco calls should be executed that are necessary for the test
type TestFunc func(fs afero.Fs, ctx TestContext)

type Option func(options *testOptions)
type Options []Option

func WithFs(fs afero.Fs) Option {
	return func(options *testOptions) {
		options.fs = fs
	}
}

// WithManifestPath is used to set the relative path of the manifest based from the current working directory of the test
func WithManifestPath(manifestPath string) Option {
	return func(options *testOptions) {
		options.manifestPath = manifestPath
	}
}

// WithManifest is used to set the manifest's name, relative to the passed configFolder.
//
//	(defaults to `{configFolder}/manifest.yaml`).
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
		if options.envVars == nil {
			options.envVars = map[string]string{}
		}

		for k, v := range env {
			options.envVars[k] = v
		}
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
//   - cleans up configurations. This can be overwritten using [WithoutCleanup]. The cleanup runs immediately after the test function fn finished.
//   - uses the suffix in the form of `<original name>_<current timestamp><defined suffix>`. The <defined suffix> can be overwritten using [WithSuffix].
//   - does not provide additional environment variables. You can use [WithEnvVars], [WithEnvVar].
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

func runIntegration(t *testing.T, opts testOptions, testFunc TestFunc) {
	configFolder, _ := filepath.Abs(opts.configFolder)

	suffix := appendUniqueSuffixToIntegrationTestConfigs(t, opts.fs, configFolder, opts.suffix)

	for k, v := range opts.envVars {
		setTestEnvVar(t, k, v, suffix)
	}

	if !opts.skipCleanup {
		defer func() {
			t.Log("Starting cleanup")
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
