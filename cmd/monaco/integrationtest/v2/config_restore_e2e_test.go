//go:build download_restore

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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/integrationtest/utils/monaco"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/featureflags"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/testutils"
)

type downloadFunction func(*testing.T, afero.Fs, string, string, string, string, bool) error

// TestRestoreConfigs validates if the configurations can be restore from the downloaded version after being deleted
// It has 5 stages:
// Preparation: Uploads a set of configurations and return the virtual filesystem
// Execution: Download the configurations to the virtual filesystem
// Cleanup: Deletes the configurations that were uploaded during validation
// Validation: Uploads the downloaded configs and checks for status code 0 as result
// Cleanup: Deletes the configurations that were uploaded during validation

// TestRestoreConfigs_FromDownloadWithManifestFile deploys, download and re-deploys from download the download-configs test-resources
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithManifestFile(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, false, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithPlatformManifestFile works like TestRestoreConfigs_FromDownloadWithManifestFile but
// has a platform environment defined in the used manifest, rather than a Classic env.
func TestRestoreConfigs_FromDownloadWithPlatformManifestFile(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, false, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithCLIParameters deploys, download and re-deploys from download the download-configs test-resources
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithCLIParameters(t *testing.T) {
	if isHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, false, execution_downloadConfigsWithCLIParameters)
}

func TestRestoreConfigs_FromDownloadWithPlatformWithCLIParameters(t *testing.T) {
	if isHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "platform_manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, true, execution_downloadConfigsWithCLIParameters)
}

func TestRestoreConfigs_FromDownloadWithPlatformManifestFile_withPlatformConfigs(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs-platform/"
	manifestFile := initialConfigsFolder + "platform_manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_automations"

	t.Setenv(featureflags.Segments.EnvName(), "true")

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, false, execution_downloadConfigs)
}

func TestDownloadWithSpecificAPIsAndSettings(t *testing.T) {

	if isHardeningEnvironment() {
		t.Skip("Skipping test as we can't set tokenEndpoint as a CLI parameter")
	}

	configsFolder, _ := filepath.Abs("test-resources/download-with-flags")
	configsFolderManifest := filepath.Join(configsFolder, "manifest.yaml")

	downloadFolder, _ := filepath.Abs("test-resources/download")

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
			RunIntegrationWithCleanup(t, configsFolder, configsFolderManifest, "", "", func(fs afero.Fs, _ TestContext) {
				err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s", configsFolderManifest))
				require.NoError(t, err)

				t.Log("Downloading configs")
				err = tc.downloadFunc(t, tc.fs, downloadFolder, tc.manifest, tc.apisToDownload, tc.settingsToDownload, false)
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
// TestRestoreConfigsFull deploys, download and re-deploys from download the all-configs test-resources
// As this downloads all configs from all APIs other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigsFull(t *testing.T) {
	t.Skipf("Test skipped as not all configurations can currently successfully be re-uploaded automatically after download")

	initialConfigsFolder := "test-resources/integration-all-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "all" // value only for testing
	suffixTest := "_download_all"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, false, execution_downloadConfigs)
}

func testRestoreConfigs(t *testing.T, initialConfigsFolder string, downloadFolder string, suffixTest string, manifestFile string, apisToDownload string, oauthEnabled bool, downloadFunc downloadFunction) {
	initialConfigsFolder, _ = filepath.Abs(initialConfigsFolder)
	downloadFolder, _ = filepath.Abs(downloadFolder)
	manifestFile, _ = filepath.Abs(manifestFile)

	fs := testutils.CreateTestFileSystem()
	suffix, err := preparation_uploadConfigs(t, fs, suffixTest, initialConfigsFolder, manifestFile)

	assert.NoError(t, err, "Error during download preparation stage")

	err = downloadFunc(t, fs, downloadFolder, manifestFile, apisToDownload, "", oauthEnabled)
	assert.NoError(t, err, "Error during download execution stage")

	integrationtest.CleanupIntegrationTest(t, fs, manifestFile, "", suffix) // remove previously deployed configs

	downloadedManifestPath := filepath.Join(downloadFolder, "manifest.yaml")

	t.Cleanup(func() { // cleanup uploaded configs after test run
		integrationtest.CleanupIntegrationTest(t, fs, manifestFile, "", suffix)
	})

	validation_uploadDownloadedConfigs(t, fs, downloadFolder, downloadedManifestPath) // re-deploy from download
}

func preparation_uploadConfigs(t *testing.T, fs afero.Fs, suffixTest string, configFolder string, manifestFile string) (suffix string, err error) {
	log.Info("BEGIN PREPARATION PROCESS")
	suffix = appendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

	// update all env values to include the _suffix suffix so that we can set env-values in configs
	for _, e := range os.Environ() {
		splits := strings.SplitN(e, "=", 2)
		key := splits[0]
		val := splits[1]

		newKey := fmt.Sprintf("%s_%s", key, suffix)

		t.Setenv(newKey, val)
	}

	t.Cleanup(func() { // register extra cleanup in case test fails after deployment
		integrationtest.CleanupIntegrationTest(t, fs, manifestFile, "", suffix)
	})

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
	oauth bool,
) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	parameters := []string{"download", "--verbose", "--output-folder", downloadFolder}
	command := fmt.Sprintf("monaco download --verbose --output-folder=%s", downloadFolder)
	if apiToDownload != "all" {
		if apiToDownload != "" {
			parameters = append(parameters, "--api", apiToDownload)
			command += " --api=" + apiToDownload
		}
		if settingToDownload != "" {
			parameters = append(parameters, "--settings-schema", settingToDownload)
			command += " --settings-schema=" + settingToDownload
		}
	}

	if oauth {
		parameters = append(parameters, "--url", os.Getenv("PLATFORM_URL_ENVIRONMENT_1"), "--token", "TOKEN_ENVIRONMENT_1", "--oauth-client-id", "OAUTH_CLIENT_ID", "--oauth-client-secret", "OAUTH_CLIENT_SECRET")
		command += fmt.Sprintf(" --url=%s --token=%s --oauth-client-id=%s --oauth-client-secret=%s", os.Getenv("PLATFORM_URL_ENVIRONMENT_1"), "TOKEN_ENVIRONMENT_1", "OAUTH_CLIENT_ID", "OAUTH_CLIENT_SECRET")
	} else {
		parameters = append(parameters, "--url", os.Getenv("URL_ENVIRONMENT_1"), "--token", "TOKEN_ENVIRONMENT_1")
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
	_ bool,
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
		log.Info("file " + fpath)
		return nil
	})

	err := monaco.Run(t, fs, fmt.Sprintf("monaco deploy %s --verbose", manifestFile))
	assert.NoError(t, err)
}
