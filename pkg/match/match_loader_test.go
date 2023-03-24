//go:build unit

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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

package match

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
)

const workingDirPath = `/home/test/monaco/match`
const name = `match-test`
const matchType = `entities`
const manifestFileName = `manifest.yaml`
const projectEnvName = `projectEnvName`
const matchFileName = "match.yaml"
const tenantUrl = "https://xxxnnnnn.live.Dynatrace.com"
const tokenType = "environment"
const tokenName = "TEST_TOKEN"
const tokenValue = "TEST"
const groupName = "default"

var workingDir = filepath.FromSlash(workingDirPath)
var outputPath = filepath.Join(workingDir, `output`)
var sourceManifestDir = filepath.Join(workingDir, `source`)
var sourceManifestPath = filepath.Join(sourceManifestDir, manifestFileName)
var targetManifestDir = filepath.Join(workingDir, `target`)
var targetManifestPath = filepath.Join(targetManifestDir, manifestFileName)
var matchFilePath = filepath.Join(workingDir, matchFileName)

const rawMatchYAMLContent = `name: %s
type: %s
outputPath: %s
sourceInfo:
  manifestPath: %s
  project: %s
  environment: %s
targetInfo:
  manifestPath: %s
  project: %s
  environment: %s
`

const rawManifestYAMLContent = `
manifestVersion: "1.0"
projects:
- name: %s
environmentGroups:
- name: %s
  environments:
  - name: %s
    url:
      value: %s
    auth:
      token:
        type: %s
        name: %s
`

func genTestMatchParameters() MatchParameters {
	return MatchParameters{
		Name:       name,
		Type:       matchType,
		WorkingDir: workingDirPath,
		OutputDir:  outputPath,
		SelfMatch:  false,
		Source: MatchParametersEnv{
			EnvType:     SOURCE_ENV,
			WorkingDir:  sourceManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					projectEnvName: manifest.ProjectDefinition{
						Name: projectEnvName,
						Path: projectEnvName,
					},
				},
				Environments: manifest.Environments{
					projectEnvName: manifest.EnvironmentDefinition{
						Name: projectEnvName,
						Type: 0,
						URL: manifest.URLDefinition{
							Type:  0,
							Value: tenantUrl,
						},
						Group: groupName,
						Auth: manifest.Auth{
							Token: manifest.AuthSecret{
								Name:  tokenName,
								Value: tokenValue,
							},
						},
					},
				},
			},
		},
		Target: MatchParametersEnv{
			EnvType:     TARGET_ENV,
			WorkingDir:  targetManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest: manifest.Manifest{
				Projects: manifest.ProjectDefinitionByProjectID{
					projectEnvName: manifest.ProjectDefinition{
						Name: projectEnvName,
						Path: projectEnvName,
					},
				},
				Environments: manifest.Environments{
					projectEnvName: manifest.EnvironmentDefinition{
						Name: projectEnvName,
						Type: 0,
						URL: manifest.URLDefinition{
							Type:  0,
							Value: tenantUrl,
						},
						Group: groupName,
						Auth: manifest.Auth{
							Token: manifest.AuthSecret{
								Name:  tokenName,
								Value: tokenValue,
							},
						},
					},
				},
			},
		},
	}
}

func TestLoadMatchingParametersFails(t *testing.T) {

	matchFileContent := fmt.Sprintf(rawMatchYAMLContent,
		name, matchType, outputPath,
		sourceManifestPath, projectEnvName, projectEnvName,
		targetManifestPath, projectEnvName, projectEnvName)

	workingDir := filepath.FromSlash(workingDirPath)
	matchFileName := "match.yaml"
	matchFilePath := filepath.Join(workingDir, matchFileName)

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, matchFilePath, []byte(matchFileContent), 0666)
	assert.NilError(t, err)

	got, err := LoadMatchingParameters(fs, matchFilePath)

	assert.Error(t, err, "Could not load Config Parameters, see errors for details")

	want := MatchParameters{
		Name:       name,
		Type:       matchType,
		WorkingDir: workingDirPath,
		OutputDir:  outputPath,
		SelfMatch:  false,
		Source: MatchParametersEnv{
			EnvType:     SOURCE_ENV,
			WorkingDir:  sourceManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest:    manifest.Manifest{},
		},
		Target: MatchParametersEnv{
			EnvType:     TARGET_ENV,
			WorkingDir:  targetManifestDir,
			Project:     projectEnvName,
			Environment: projectEnvName,
			Manifest:    manifest.Manifest{},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadMatchingParameters() got = %v, want %v", got, want)
	}
}

func TestLoadMatchingParameters(t *testing.T) {

	t.Setenv(tokenName, tokenValue)

	matchFileContent := fmt.Sprintf(rawMatchYAMLContent,
		name, matchType, outputPath,
		sourceManifestPath, projectEnvName, projectEnvName,
		targetManifestPath, projectEnvName, projectEnvName)

	fs := afero.NewMemMapFs()
	err := fs.MkdirAll(workingDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, matchFilePath, []byte(matchFileContent), 0666)
	assert.NilError(t, err)

	sourceManifestFileContent := fmt.Sprintf(rawManifestYAMLContent,
		projectEnvName, groupName, projectEnvName, tenantUrl, tokenType, tokenName)

	err = fs.MkdirAll(sourceManifestDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, sourceManifestPath, []byte(sourceManifestFileContent), 0666)
	assert.NilError(t, err)

	targetManifestFileContent := fmt.Sprintf(rawManifestYAMLContent,
		projectEnvName, groupName, projectEnvName, tenantUrl, tokenType, tokenName)

	err = fs.MkdirAll(targetManifestDir, 0777)

	assert.NilError(t, err)

	err = afero.WriteFile(fs, targetManifestPath, []byte(targetManifestFileContent), 0666)
	assert.NilError(t, err)

	got, err := LoadMatchingParameters(fs, matchFilePath)

	assert.NilError(t, err)

	want := genTestMatchParameters()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadMatchingParameters() got = %v, want %v", got, want)
	}
}
