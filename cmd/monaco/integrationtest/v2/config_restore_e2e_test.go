//go:build integration
// +build integration

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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/runner"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/manifest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"github.com/spf13/afero"
	"gotest.tools/assert"
)

type downloadFunction func(*testing.T, afero.Fs, string, string, string) error

//TestRestoreConfigs validates if the configurations can be restore from the downloaded version after being deleted
//It has 5 stages:
//Preparation: Uploads a set of configurations and return the virtual filesystem
//Execution: Download the configurations to the virtual filesystem
//Cleanup: Deletes the configurations that were uploaded during validation
//Validation: Uploads the downloaded configs and checks for status code 0 as result
//Cleanup: Deletes the configurations that were uploaded during validation

// This version runs the test against 2 simple configs (alerting profiles and management zones)
func TestRestoreConfigs_FromDownloadWithManifestFile(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_manifest"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, execution_downloadConfigs)
}

func TestRestoreConfigs_FromDownloadWithCLIParameters(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
	manifestFile := initialConfigsFolder + "manifest.yaml"
	downloadFolder := "test-resources/download"
	subsetOfConfigsToDownload := "alerting-profile,management-zone"
	suffixTest := "_download_cli-only"

	testRestoreConfigs(t, initialConfigsFolder, downloadFolder, suffixTest, manifestFile, subsetOfConfigsToDownload, execution_downloadConfigsWithCLIParameters)
}

// This version runs the test against the all_configs project
func TestRestoreConfigsFull(t *testing.T) {
	initialConfigsFolder := "test-resources/integration-download-configs/"
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

	fs := util.CreateTestFileSystem()
	suffix, err := preparation_uploadConfigs(t, fs, suffixTest, initialConfigsFolder, manifestFile)

	assert.NilError(t, err, "Error during download preparation stage")

	err = downloadFunc(t, fs, downloadFolder, manifestFile, apisToDownload)
	assert.NilError(t, err, "Error during download execution stage")

	cleanupDeployedConfiguration(t, fs, manifestFile, suffix) // remove previously deployed configs

	downloadedManifestPath := filepath.Join(downloadFolder, "manifest.yaml")
	validation_uploadDownloadedConfigs(t, fs, downloadFolder, downloadedManifestPath) // re-deploy from download

	cleanupDeployedConfiguration(t, fs, filepath.Join(downloadFolder, "manifest.yaml"), suffix) // cleanup
}

func preparation_uploadConfigs(t *testing.T, fs afero.Fs, suffixTest string, configFolder string, manifestFile string) (suffix string, err error) {
	log.Info("BEGIN PREPARATION PROCESS")
	suffix = appendUniqueSuffixToIntegrationTestConfigs(t, fs, configFolder, suffixTest)

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
	apisToDownload string) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	parameters := []string{}

	if apisToDownload == "all" {
		parameters = []string{
			"download",
			"--verbose",
			"--url", os.Getenv("URL_ENVIRONMENT_1"),
			"--token", "TOKEN_ENVIRONMENT_1",
			"--output-folder", downloadFolder,
		}
	} else {
		parameters = []string{
			"download",
			"--verbose",
			"--specific-api",
			apisToDownload,
			"--url", os.Getenv("URL_ENVIRONMENT_1"),
			"--token", "TOKEN_ENVIRONMENT_1",
			"--output-folder", downloadFolder,
		}
	}

	cmd := runner.BuildCli(fs)
	cmd.SetArgs(parameters)
	err = cmd.Execute()
	assert.NilError(t, err)
	return nil
}

func execution_downloadConfigs(t *testing.T, fs afero.Fs, downloadFolder string, manifestFile string,
	apisToDownload string) error {
	log.Info("BEGIN DOWNLOAD PROCESS")

	downloadFolder, err := filepath.Abs(downloadFolder)
	if err != nil {
		return err
	}
	parameters := []string{}

	if apisToDownload == "all" {
		parameters = []string{
			"download",
			"--verbose",
			"--manifest", manifestFile,
			"--specific-environment", "environment1",
			"--output-folder", downloadFolder,
		}
	} else {
		parameters = []string{
			"download",
			"--verbose",
			"--specific-api",
			apisToDownload,
			"--manifest", manifestFile,
			"--specific-environment", "environment1",
			"--output-folder", downloadFolder,
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
	FailOnAnyError(errs, "loading of manifest failed")

	cleanupIntegrationTest(t, loadedManifest, "", testSuffix)
}
