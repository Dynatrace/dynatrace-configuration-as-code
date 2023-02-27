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
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/integrationtest"
	"github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/runner"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/testutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"
)

type downloadFunction func(*testing.T, afero.Fs, string, string, string, string) error

//TestRestoreConfigs validates if the configurations can be restore from the downloaded version after being deleted
//It has 5 stages:
//Preparation: Uploads a set of configurations and return the virtual filesystem
//Execution: Download the configurations to the virtual filesystem
//Cleanup: Deletes the configurations that were uploaded during validation
//Validation: Uploads the downloaded configs and checks for status code 0 as result
//Cleanup: Deletes the configurations that were uploaded during validation

// TestRestoreConfigs_FromDownloadWithManifestFile deploys, download and re-deploys from download the download-configs test-resources
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithManifestFile(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, execution_downloadConfigs)
}

// TestRestoreConfigs_FromDownloadWithCLIParameters deploys, download and re-deploys from download the download-configs test-resources
// As this downloads all alerting-profile and management-zone configs, other tests and their cleanup are likely to interfere
// Thus download_restore tests should be run independently to other integration tests
func TestRestoreConfigs_FromDownloadWithCLIParameters(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, execution_downloadConfigsWithCLIParameters)
}

func TestDownloadWithSpecificAPIsAndSettings(t *testing.T) {
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

	for _, tt := range tests {
		RunIntegrationWithCleanup(t, configsFolder, configsFolderManifest, "", tt.name, func(fs afero.Fs) {
			t.Run(tt.name, func(t *testing.T) {

				t.Log("Deploying configs")
				cmd := runner.BuildCli(fs)
				cmd.SetArgs([]string{"deploy", "-v", configsFolderManifest})
				err := cmd.Execute()

				t.Log("Downloading configs")
				err = tt.downloadFunc(t, tt.fs, downloadFolder, tt.manifest, tt.apisToDownload, tt.settingsToDownload)
				assert.Equal(t, tt.wantErr, err != nil)
				for _, f := range tt.expectedFolders {
					folderExists, _ := afero.DirExists(tt.fs, f)
					assert.Check(t, folderExists, "folder "+f+" does not exist")
				}
				files, _ := afero.ReadDir(tt.fs, tt.projectFolder)
				assert.Equal(t, len(tt.expectedFolders), len(files))
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
	subsetOfConfigsToDownload := "all" //value only for testing
	suffixTest := "_download_all"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, execution_downloadConfigs)
}

func testRestoreConfigs(t *testing.T, initialConfigsFolder string, downloadFolder string, suffixTest string, manifestFile string, apisToDownload string, downloadFunc downloadFunction) {
	initialConfigsFolder, _ = filepath.Abs(initialConfigsFolder)
	downloadFolder, _ = filepath.Abs(downloadFolder)
	manifestFile, _ = filepath.Abs(manifestFile)

	fs := testutils.CreateTestFileSystem()
	suffix, err := preparation_uploadConfigs(t, fs, suffixTest, initialConfigsFolder, manifestFile)

	assert.NilError(t, err, "Error during download preparation stage")

	err = downloadFunc(t, fs, downloadFolder, manifestFile, apisToDownload, "")
	assert.NilError(t, err, "Error during download execution stage")

	cleanupDeployedConfiguration(t, fs, manifestFile, suffix) // remove previously deployed configs

	downloadedManifestPath := filepath.Join(downloadFolder, "manifest.yaml")

	t.Cleanup(func() { // cleanup uploaded configs after test run
		cleanupDeployedConfiguration(t, fs, downloadedManifestPath, suffix)
	})

	validation_uploadDownloadedConfigs(t, fs, downloadFolder, downloadedManifestPath) // re-deploy from download
}

func preparation_uploadConfigs(t *testing.T, fs afero.Fs, suffixTest string, configFolder string, manifestFile string) (suffix string, err error) {
	log.Info("BEGIN PREPARATION PROCESS")
	suffix = appendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

	t.Cleanup(func() { // register extra cleanup in case test fails after deployment
		cleanupDeployedConfiguration(t, fs, manifestFile, suffix)
	})

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		manifestFile,
	})
	err = cmd.Execute()
	assert.NilError(t, err)

	return suffix, nil
}

func execution_downloadConfigsWithCLIParameters(t *testing.T, fs afero.Fs, downloadFolder string, _ string,
	apisToDownload string, settingsToDownload string) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	var parameters []string
	if apisToDownload == "all" {
		parameters = []string{
			"download",
			"direct",
			os.Getenv("URL_ENVIRONMENT_1"),
			"TOKEN_ENVIRONMENT_1",
			"--verbose",
			"--output-folder", downloadFolder,
		}
	} else {
		parameters = []string{
			"download",
			"direct",
			os.Getenv("URL_ENVIRONMENT_1"),
			"TOKEN_ENVIRONMENT_1",
			"--verbose",
			"--output-folder", downloadFolder,
		}
		if apisToDownload != "" {
			parameters = append(parameters, "--api", apisToDownload)
		}
		if settingsToDownload != "" {
			parameters = append(parameters, "--settings-schema", settingsToDownload)
		}
	}

	cmd := runner.BuildCli(fs)
	cmd.SetArgs(parameters)
	err = cmd.Execute()
	assert.NilError(t, err)
	return nil
}

func execution_downloadConfigs(t *testing.T, fs afero.Fs, downloadFolder string, manifestFile string,
	apisToDownload string, settingsToDownload string) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	parameters := []string{}

	if apisToDownload == "all" {
		parameters = []string{
			"download",
			"manifest",
			manifestFile,
			"environment1",
			"--verbose",
			"--output-folder", downloadFolder,
		}
	} else {
		parameters = []string{
			"download",
			"manifest",
			manifestFile,
			"environment1",
			"--verbose",
			"--output-folder", downloadFolder,
		}

		if apisToDownload != "" {
			parameters = append(parameters, "--api", apisToDownload)
		}
		if settingsToDownload != "" {
			parameters = append(parameters, "--settings-schema", settingsToDownload)
		}
	}

	cmd := runner.BuildCli(fs)
	cmd.SetArgs(parameters)
	err = cmd.Execute()
	assert.NilError(t, err)
	return nil
}

func validation_uploadDownloadedConfigs(t *testing.T, fs afero.Fs, downloadFolder string,
	manifestFile string) {
	log.Info("BEGIN VALIDATION PROCESS")
	//Shows you the downloaded files list in the command line
	_ = afero.Walk(fs, downloadFolder+"/", func(path string, info os.FileInfo, err error) error {
		fpath, err := filepath.Abs(path)
		log.Info("file " + fpath)
		return nil
	})

	cmd := runner.BuildCli(fs)
	cmd.SetArgs([]string{
		"deploy",
		"--verbose",
		manifestFile,
	})
	err := cmd.Execute()
	assert.NilError(t, err)
}

func cleanupDeployedConfiguration(t *testing.T, fs afero.Fs, manifestFilepath string, testSuffix string) {
	loadedManifest, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestFilepath,
	})
	testutils.FailTestOnAnyError(t, errs, "loading of manifest failed")

	integrationtest.CleanupIntegrationTest(t, fs, manifestFilepath, loadedManifest, "", testSuffix)
}
