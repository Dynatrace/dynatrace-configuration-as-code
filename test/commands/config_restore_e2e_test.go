//go:build download_restore

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/test/internal/runner"
)

type AuthType int

const (
	AuthClassicToken AuthType = iota
	AuthOAuth
	AuthPlatformToken
)

type downloadFunction func(*testing.T, afero.Fs, string, string, string, string, AuthType) error

// TestRestoreConfigs validates if the configurations can be restored from the downloaded version after being deleted
// It has 5 stages:
// Preparation: Uploads a set of configurations and return the virtual filesystem
// Execution: Download the configurations to the virtual filesystem
// Cleanup: Deletes the configurations that were uploaded during validation
// Validation: Uploads the downloaded configs and checks for status code 0 as result
// Cleanup: Deletes the configurations that were uploaded during validation

// TestRestoreConfigs_FromDownloadWithManifestFile deploys, download and re-deploys from download the download-configs testdata
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithManifestFile(t *testing.T) {
	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthClassicToken, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithPlatformOAuthManifestFile works like TestRestoreConfigs_FromDownloadWithManifestFile but
// has a platform environment with OAuth credentials defined in the used manifest, rather than a Classic env.
func TestRestoreConfigs_FromDownloadWithPlatformOAuthManifestFile(t *testing.T) {
	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_oauth_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthOAuth, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithPlatformTokenManifestFile works like
// TestRestoreConfigs_FromDownloadWithPlatformOAuthManifestFile but has a platform token defined in the used manifest,
// rather than OAuth credentials.
func TestRestoreConfigs_FromDownloadWithPlatformTokenManifestFile(t *testing.T) {
	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_token_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	t.Setenv(featureflags.PlatformToken.EnvName(), "true")

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthPlatformToken, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithCLIParameters deploys, download and re-deploys from download the download-configs testdata
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithCLIParameters(t *testing.T) {
	if runner.IsHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthClassicToken, execution_downloadConfigsWithCLIParameters)
}

func TestRestoreConfigs_FromDownloadWithPlatformOAuthWithCLIParameters(t *testing.T) {
	if runner.IsHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_oauth_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthOAuth, execution_downloadConfigsWithCLIParameters)
}

func TestRestoreConfigs_FromDownloadWithPlatformTokenWithCLIParameters(t *testing.T) {
	if runner.IsHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	t.Setenv(featureflags.PlatformToken.EnvName(), "true")

	initialConfigsFolder := "testdata/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_token_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthPlatformToken, execution_downloadConfigsWithCLIParameters)
}

func TestRestoreConfigs_FromDownloadWithPlatformOAuthManifestFile_withPlatformConfigs(t *testing.T) {
	initialConfigsFolder := "testdata/integration-download-configs-platform/"
	manifestFile := initialConfigsFolder + "platform_oauth_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_automations"

	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthOAuth, execution_downloadConfigs)
}

func TestRestoreConfigs_FromDownloadWithPlatformTokenManifestFile_withPlatformConfigs(t *testing.T) {
	initialConfigsFolder := "testdata/integration-download-configs-platform/"
	manifestFile := initialConfigsFolder + "platform_token_manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_automations"

	t.Setenv(featureflags.ServiceLevelObjective.EnvName(), "true")
	t.Setenv(featureflags.PlatformToken.EnvName(), "true")

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthPlatformToken, execution_downloadConfigs)
}

func TestDownloadWithSpecificAPIsAndSettings(t *testing.T) {

	if runner.IsHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	configsFolder, _ := filepath.Abs("testdata/download-with-flags")
	configsFolderManifest := filepath.Join(configsFolder, "manifest.yaml")

	downloadFolder, _ := filepath.Abs("testdata/download")

	tests := []struct {
		name               string
		fs                 afero.Fs
		downloadFunc       downloadFunction
		apisToDownload     string // comma separated list of apis
		settingsToDownload string // comma setting list of schema IDs
		projectFolder      string
		manifest           string
		expectedFolders    []string
		wantErr            bool
	}{
		{
			name:               "using --api and --settings-schema",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigsWithCLIParameters,
			projectFolder:      downloadFolder + "/project",
			apisToDownload:     "auto-tag",
			settingsToDownload: "builtin:alerting.profile",
			expectedFolders: []string{
				downloadFolder + "/project/auto-tag",
				downloadFolder + "/project/builtinalerting.profile"},
			wantErr: false,
		},
		{
			name:               "using --api",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigsWithCLIParameters,
			projectFolder:      downloadFolder + "/project",
			apisToDownload:     "auto-tag",
			settingsToDownload: "",
			expectedFolders: []string{
				downloadFolder + "/project/auto-tag"},
			wantErr: false,
		},
		{
			name:               "using --settings-schema",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigsWithCLIParameters,
			projectFolder:      downloadFolder + "/project",
			apisToDownload:     "",
			settingsToDownload: "builtin:alerting.profile",
			expectedFolders: []string{
				downloadFolder + "/project/builtinalerting.profile"},
			wantErr: false,
		},
		{
			name:               "using --api and --settings-schema (manifest)",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigs,
			projectFolder:      downloadFolder + "/project_environment1",
			manifest:           configsFolderManifest,
			apisToDownload:     "auto-tag",
			settingsToDownload: "builtin:alerting.profile",
			expectedFolders: []string{
				downloadFolder + "/project_environment1/auto-tag",
				downloadFolder + "/project_environment1/builtinalerting.profile"},
			wantErr: false,
		},
		{
			name:               "using --api (manifest)",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigs,
			projectFolder:      downloadFolder + "/project_environment1",
			manifest:           configsFolderManifest,
			apisToDownload:     "auto-tag",
			settingsToDownload: "",
			expectedFolders: []string{
				downloadFolder + "/project_environment1/auto-tag"},
			wantErr: false,
		},
		{
			name:               "using --specific-settings (manifest)",
			fs:                 testutils.CreateTestFileSystem(),
			downloadFunc:       execution_downloadConfigs,
			projectFolder:      downloadFolder + "/project_environment1",
			manifest:           configsFolderManifest,
			apisToDownload:     "",
			settingsToDownload: "builtin:alerting.profile",
			expectedFolders: []string{
				downloadFolder + "/project_environment1/builtinalerting.profile"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner.Run(t, configsFolder,
				runner.Options{
					runner.WithManifestPath(configsFolderManifest),
					runner.WithSuffix(""),
				},
				func(fs afero.Fs, _ runner.TestContext) {
					err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s", configsFolderManifest))
					require.NoError(t, err)

					t.Log("Downloading configs")
					err = tc.downloadFunc(t, tc.fs, downloadFolder, tc.manifest, tc.apisToDownload, tc.settingsToDownload, AuthClassicToken)
					assert.Equal(t, tc.wantErr, err != nil)
					for _, f := range tc.expectedFolders {
						folderExists, _ := afero.DirExists(tc.fs, f)
						assert.Truef(t, folderExists, "folder %s does not exist", f)
					}
					files, _ := afero.ReadDir(tc.fs, tc.projectFolder)
					assert.Equal(t, len(tc.expectedFolders), len(files))
				})
		})
	}
}

// TestRestoreConfigsFull is currently
// TestRestoreConfigsFull deploys, download and re-deploys from download the all-configs testdata
// As this downloads all configs from all APIs other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigsFull(t *testing.T) {
	t.Skipf("Test skipped as not all configurations can currently successfully be re-uploaded automatically after download")

	initialConfigsFolder := "testdata/integration-all-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "testdata/download"
	subsetOfConfigsToDownload := "all" // value only for testing
	suffixTest := "_download_all"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, AuthClassicToken, execution_downloadConfigs)
}

func testRestoreConfigs(t *testing.T, initialConfigsFolder string, downloadFolder string, suffixTest string, manifestFile string, apisToDownload string, authType AuthType, downloadFunc downloadFunction) {
	initialConfigsFolder, _ = filepath.Abs(initialConfigsFolder)
	downloadFolder, _ = filepath.Abs(downloadFolder)
	manifestFile, _ = filepath.Abs(manifestFile)

	fs := testutils.CreateTestFileSystem()
	suffix, err := preparation_uploadConfigs(t, fs, suffixTest, initialConfigsFolder, manifestFile)

	assert.NoError(t, err, "Error during download preparation stage")

	err = downloadFunc(t, fs, downloadFolder, manifestFile, apisToDownload, "", authType)
	assert.NoError(t, err, "Error during download execution stage")

	runner.CleanupIntegrationTest(t, fs, manifestFile, "", suffix) // remove previously deployed configs

	downloadedManifestPath := filepath.Join(downloadFolder, "manifest.yaml")

	defer func() { // cleanup uploaded configs after test run
		runner.CleanupIntegrationTest(t, fs, manifestFile, "", suffix)
	}()

	validation_uploadDownloadedConfigs(t, fs, downloadFolder, downloadedManifestPath) // re-deploy from download
}

func preparation_uploadConfigs(t *testing.T, fs afero.Fs, suffixTest string, configFolder string, manifestFile string) (suffix string, err error) {
	log.Info("BEGIN PREPARATION PROCESS")
	suffix = runner.AppendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

	// update all env values to include the _suffix suffix so that we can set env-values in configs
	for _, e := range os.Environ() {
		splits := strings.SplitN(e, "=", 2)
		key := splits[0]
		val := splits[1]

		newKey := fmt.Sprintf("%s_%s", key, suffix)

		t.Setenv(newKey, val)
	}

	defer func() { // register extra cleanup in case test fails after deployment
		runner.CleanupIntegrationTest(t, fs, manifestFile, "", suffix)
	}()

	err = monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifestFile))
	assert.NoError(t, err)

	return suffix, nil
}

func execution_downloadConfigsWithCLIParameters(
	t *testing.T,
	fs afero.Fs,
	downloadFolder string,
	_ string,
	apiToDownload string,
	settingToDownload string,
	authType AuthType,
) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	command := fmt.Sprintf("monaco download --verbose --output-folder=%s", downloadFolder)
	if apiToDownload != "all" {
		if apiToDownload != "" {
			command += " --api=" + apiToDownload
		}
		if settingToDownload != "" {
			command += " --settings-schema=" + settingToDownload
		}
	}

	if authType == AuthOAuth {
		command += fmt.Sprintf(" --url=%s --token=%s --oauth-client-id=%s --oauth-client-secret=%s", os.Getenv("PLATFORM_URL_ENVIRONMENT_1"), "TOKEN_ENVIRONMENT_1", "OAUTH_CLIENT_ID", "OAUTH_CLIENT_SECRET")
	} else if authType == AuthPlatformToken {
		command += fmt.Sprintf(" --url=%s --token=%s --platform-token=%s", os.Getenv("PLATFORM_URL_ENVIRONMENT_1"), "TOKEN_ENVIRONMENT_1", "PLATFORM_TOKEN")
	} else if authType == AuthClassicToken {
		command += fmt.Sprintf(" --url=%s --token=%s", os.Getenv("URL_ENVIRONMENT_1"), "TOKEN_ENVIRONMENT_1")
	}

	err = monaco.Run(t, fs, command)
	assert.NoError(t, err)
	return nil
}

func execution_downloadConfigs(
	t *testing.T,
	fs afero.Fs,
	downloadFolder string,
	manifestFile string,
	apisToDownload string,
	settingsToDownload string,
	_ AuthType,
) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	var command string

	if apisToDownload == "all" {
		command = fmt.Sprintf("monaco download --manifest=%s --environment=environment1 --verbose --output-folder=%s", manifestFile, downloadFolder)
	} else {
		command = fmt.Sprintf("monaco download --manifest=%s --output-folder=%s --environment=environment1 --verbose", manifestFile, downloadFolder)
		if apisToDownload != "" {
			command += fmt.Sprintf(" --api=%s", apisToDownload)
		}
		if settingsToDownload != "" {
			command += fmt.Sprintf(" --settings-schema=%s", settingsToDownload)
		}
	}

	err = monaco.Run(t, fs, command)
	assert.NoError(t, err)
	return nil
}

func validation_uploadDownloadedConfigs(t *testing.T, fs afero.Fs, downloadFolder string,
	manifestFile string) {
	log.Info("BEGIN VALIDATION PROCESS")
	// Shows you the downloaded files list in the command line
	_ = afero.Walk(fs, downloadFolder+"/", func(path string, info os.FileInfo, err error) error {
		fpath, err := filepath.Abs(path)
		log.Info("file %s", fpath)
		return nil
	})

	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifestFile))
	assert.NoError(t, err)
}
