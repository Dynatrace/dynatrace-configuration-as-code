// @license
// Copyright 2022 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build unit

package download

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter/value"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/manifest"
	v2 "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/project/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

const (
	manifestWithEnvVarUrl string = `manifestVersion: "1.0"
projects:
- name: test-project
environmentGroups:
- name: default
  environments:
  - name: test-project
    url:
      type: environment
      value: ENVIRONMENT_URL
    auth:
      token:
        type: environment
        name: TEST_ENV_TOKEN
`
	manifestWithValueUrl string = `manifestVersion: "1.0"
projects:
- name: test-project
environmentGroups:
- name: default
  environments:
  - name: test-project
    url:
      value: env.url.com
    auth:
      token:
        type: environment
        name: TEST_ENV_TOKEN
`
)

func TestWriteToDisk(t *testing.T) {

	type args struct {
		fs                afero.Fs
		downloadedConfigs v2.ConfigsPerType
		projectName       string
		tokenEnvVarName   string
		environmentUrl    manifest.URLDefinition
		outputFolder      string
		timestampString   string
	}
	tests := []struct {
		name                string
		args                args
		wantOutputFolder    string
		wantManifestFile    string
		wantManifestContent string
		wantErr             bool
		forceOverwrite      bool
	}{
		{
			"creates expected files",
			args{
				fs: emptyTestFs(),
				downloadedConfigs: v2.ConfigsPerType{
					"test-api": []config.Config{
						{
							Type:        config.ClassicApiType{Api: "test-api"},
							Template:    template.NewInMemoryTemplate("template.json", "{}"),
							Coordinate:  coordinate.Coordinate{},
							Group:       "",
							Environment: "",
							Parameters: config.Parameters{
								"name": value.New("test-config"),
							},
							Skip: false,
						},
					},
				},
				projectName:     "test-project",
				tokenEnvVarName: "TEST_ENV_TOKEN",
				environmentUrl: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: "env.url.com",
				},
				outputFolder:    "test-output",
				timestampString: "TESTING_TIME",
			},
			"test-output",
			"manifest.yaml",
			manifestWithValueUrl,
			false,
			false,
		},
		{
			"creates 'download_{TIMESTAMP}' output if no output folder is defined",
			args{
				fs: emptyTestFs(),
				downloadedConfigs: v2.ConfigsPerType{
					"test-api": []config.Config{
						{
							Type:        config.ClassicApiType{Api: "test-api"},
							Template:    template.NewInMemoryTemplate("template.json", "{}"),
							Coordinate:  coordinate.Coordinate{},
							Group:       "",
							Environment: "",
							Parameters: config.Parameters{
								"name": value.New("test-config"),
							},
							Skip: false,
						},
					},
				},
				projectName:     "test-project",
				tokenEnvVarName: "TEST_ENV_TOKEN",
				environmentUrl: manifest.URLDefinition{
					Type: manifest.EnvironmentURLType,
					Name: "ENVIRONMENT_URL",
				},
				outputFolder:    "",
				timestampString: "TESTING_TIME",
			},
			"download_TESTING_TIME",
			"manifest.yaml",
			manifestWithEnvVarUrl,
			false,
			false,
		},
		{
			"creates 'manifest_{TIMESTAMP}' if a manifest.yaml already exists",
			args{
				fs: testFsWithWithExistingManifest("test-output"),
				downloadedConfigs: v2.ConfigsPerType{
					"test-api": []config.Config{
						{
							Type:        config.ClassicApiType{Api: "test-api"},
							Template:    template.NewInMemoryTemplate("template.json", "{}"),
							Coordinate:  coordinate.Coordinate{},
							Group:       "",
							Environment: "",
							Parameters: config.Parameters{
								"name": value.New("test-config"),
							},
							Skip: false,
						},
					},
				},
				projectName:     "test-project",
				tokenEnvVarName: "TEST_ENV_TOKEN",
				environmentUrl: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: "env.url.com",
				},
				outputFolder:    "test-output",
				timestampString: "TESTING_TIME",
			},
			"test-output",
			"manifest_TESTING_TIME.yaml",
			manifestWithValueUrl,
			false,
			false,
		},
		{
			"overwrites existing manifest.yaml if forced overwrite",
			args{
				fs: testFsWithWithExistingManifest("test-output"),
				downloadedConfigs: v2.ConfigsPerType{
					"test-api": []config.Config{
						{
							Type:        config.ClassicApiType{Api: "test-api"},
							Template:    template.NewInMemoryTemplate("template.json", "{}"),
							Coordinate:  coordinate.Coordinate{},
							Group:       "",
							Environment: "",
							Parameters: config.Parameters{
								"name": value.New("test-config"),
							},
							Skip: false,
						},
					},
				},
				projectName:     "test-project",
				tokenEnvVarName: "TEST_ENV_TOKEN",
				environmentUrl: manifest.URLDefinition{
					Type:  manifest.ValueURLType,
					Value: "env.url.com",
				},
				outputFolder:    "test-output",
				timestampString: "TESTING_TIME",
			},
			"test-output",
			"manifest.yaml",
			manifestWithValueUrl,
			false,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proj := CreateProjectData(tt.args.downloadedConfigs, tt.args.projectName) //using CreateProject data to simplify test struct setup
			writerContext := WriterContext{
				EnvironmentToWrite: proj,
				Auth: manifest.Auth{Token: &manifest.AuthSecret{
					Name: tt.args.tokenEnvVarName,
				}},
				EnvironmentUrl:  tt.args.environmentUrl,
				OutputFolder:    tt.args.outputFolder,
				timestampString: tt.args.timestampString,
				ForceOverwrite:  tt.forceOverwrite,
			}

			if err := writeToDisk(tt.args.fs, writerContext); (err != nil) != tt.wantErr {
				t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
			}

			if exists, err := afero.Exists(tt.args.fs, tt.args.outputFolder); err != nil || !exists {
				t.Errorf("Expected outputfolder %v was not created", tt.args.outputFolder)
			}

			if exists, err := afero.Exists(tt.args.fs, tt.wantOutputFolder); err != nil || !exists {
				t.Errorf("Expected outputfolder %v was not created", tt.wantOutputFolder)
			}

			expectedProjectFolder := filepath.Join(tt.wantOutputFolder, tt.args.projectName)
			if exists, err := afero.Exists(tt.args.fs, expectedProjectFolder); err != nil || !exists {
				t.Errorf("Expected project %v was not created", expectedProjectFolder)
			}

			expectedManifest := filepath.Join(tt.wantOutputFolder, tt.wantManifestFile)
			if exists, err := afero.Exists(tt.args.fs, expectedManifest); err != nil || !exists {
				t.Errorf("Expected manifest %v was not created", expectedManifest)
			}

			actualManifestContent, err := afero.ReadFile(tt.args.fs, expectedManifest)
			require.NoError(t, err)
			assert.Equal(t, tt.wantManifestContent, string(actualManifestContent),
				"Manifest content was expected to be:\n%v\nbut actually was:\n%v", tt.wantManifestContent, string(actualManifestContent))
		})
	}
}

func TestWriteToDisk_OverwritesManifestIfForced(t *testing.T) {
	//GIVEN
	downloadedConfigs := v2.ConfigsPerType{
		"test-api": []config.Config{
			{
				Type:        config.ClassicApiType{Api: "test-api"},
				Template:    template.NewInMemoryTemplate("template.json", "{}"),
				Coordinate:  coordinate.Coordinate{},
				Group:       "",
				Environment: "",
				Parameters: config.Parameters{
					"name": value.New("test-config"),
				},
				Skip: false,
			},
		},
	}
	projectName := "test-project"
	tokenEnvVarName := "TEST_ENV_TOKEN"
	environmentUrl := "env.url.com"
	outputFolder := "test-output"
	timestampString := "TESTING_TIME"
	proj := CreateProjectData(downloadedConfigs, projectName) //using CreateProject data to simplify test struct setup
	writerContext := WriterContext{
		EnvironmentToWrite: proj,
		Auth: manifest.Auth{Token: &manifest.AuthSecret{
			Name: tokenEnvVarName,
		}},
		EnvironmentUrl: manifest.URLDefinition{
			Type:  manifest.EnvironmentURLType,
			Value: environmentUrl,
		},
		OutputFolder:    outputFolder,
		timestampString: timestampString,
	}

	//GIVEN existing data on disk
	manifestPath := filepath.Join(outputFolder, "manifest.yaml")
	fs := testFsWithWithExistingManifest(outputFolder)
	previousManifest, err := afero.ReadFile(fs, manifestPath)
	require.NoError(t, err)

	//WHEN writing to disk with overwrite forced
	writerContext.ForceOverwrite = true
	err = writeToDisk(fs, writerContext)
	require.NoError(t, err)

	//THEN manifest.yaml is overwritten
	if exists, err := afero.Exists(fs, "test-output/manifest.yaml"); err != nil || !exists {
		t.Errorf("WriteToDisk(): expected manifest \"test-output/manifest.yaml\" to exist")
	}

	additionalManifestPath := filepath.Join(outputFolder, "manifest_TESTING_TIME.yaml")
	if exists, err := afero.Exists(fs, additionalManifestPath); err != nil || exists {
		t.Errorf("WriteToDisk(): expected no additional manifest %s to exist", additionalManifestPath)
	}

	writtenManifest, err := afero.ReadFile(fs, manifestPath)
	require.NoError(t, err)
	assert.NotEqual(t, string(writtenManifest), string(previousManifest), "Expected manifest to be overwritten with new data")

}

func emptyTestFs() afero.Fs {
	return afero.NewMemMapFs()
}

func testFsWithWithExistingManifest(folder string) afero.Fs {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll(folder, 0777)
	_ = afero.WriteFile(fs, filepath.Join(folder, "manifest.yaml"), []byte{}, 0777)
	return fs
}
