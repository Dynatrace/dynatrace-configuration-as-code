// @license
// Copyright 2021 Dynatrace LLC
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

package entities

import (
	"fmt"
	"strings"

	cmdUtil "github.com/dynatrace/dynatrace-configuration-as-code/cmd/monaco/util"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

var validMatchTypes = map[string]bool{
	"entities": true,
	"configs":  true,
}

const SOURCE_ENV = "Source"
const TARGET_ENV = "Target"

type matchLoaderContext struct {
	fs            afero.Fs
	matchFilePath string
}

type MatchParameters struct {
	Name       string
	Type       string
	WorkingDir string
	Source     MatchParametersEnv
	Target     MatchParametersEnv
}

type MatchParametersEnv struct {
	EnvType     string
	WorkingDir  string
	Project     string
	Environment string
	Manifest    manifest.Manifest
}

type MatchFileDefinition struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`
	Source EnvInfoDefinition `yaml:"sourceInfo"`
	Target EnvInfoDefinition `yaml:"targetInfo"`
}

type EnvInfoDefinition struct {
	ManifestPath string `yaml:"manifestPath"`
	Project      string `yaml:"project"`
	Environment  string `yaml:"environment"`
}

type MatchEntryParserError struct {
	Value  string
	Index  int
	Reason string
}

func LoadMatchingParameters(fs afero.Fs, matchFileName string) (matchParameters MatchParameters, err error) {
	matchWorkingDir, matchFilePath, err := cmdUtil.GetFilePaths(matchFileName)
	if err != nil {
		return
	}
	matchParameters.WorkingDir = matchWorkingDir

	context := &matchLoaderContext{
		fs:            fs,
		matchFilePath: matchFilePath,
	}

	matchFileDef, err := parseMatchFile(context)
	if err != nil {
		return
	}

	var errors []error

	if matchFileDef.Name == "" {
		errors = append(errors, fmt.Errorf("matches should be named"))
	} else {
		matchParameters.Name = matchFileDef.Name
	}

	if validMatchTypes[matchFileDef.Type] {
		matchParameters.Type = matchFileDef.Type
	} else {
		errors = append(errors, fmt.Errorf("matches type should be: %s, but was: %s", strings.Join(getMapKeys(validMatchTypes), " or "), matchFileDef.Type))
	}

	var errList []error
	matchParameters.Source, errList = getParameterEnv(context, matchFileDef.Source, SOURCE_ENV)

	if errList != nil {
		errors = append(errors, errList...)
	}

	matchParameters.Target, errList = getParameterEnv(context, matchFileDef.Target, TARGET_ENV)

	if errList != nil {
		errors = append(errors, errList...)
	}

	if len(errors) > 0 {
		err = util.PrintAndFormatErrors(errors, "Could not load Config Parameters, see errors for details")
	}

	return
}

func getParameterEnv(context *matchLoaderContext, matchInfoDef EnvInfoDefinition, envType string) (MatchParametersEnv, []error) {
	matchParametersEnv := MatchParametersEnv{}
	var err error
	var errors []error

	matchParametersEnv.Manifest, err = cmdUtil.GetManifest(context.fs, matchInfoDef.ManifestPath)
	if err != nil {
		errors = append(errors, err)
	}

	matchParametersEnv.WorkingDir, _, err = cmdUtil.GetFilePaths(matchInfoDef.ManifestPath)
	if err != nil {
		errors = append(errors, err)
	}

	matchParametersEnv.EnvType = envType
	matchParametersEnv.Project = matchInfoDef.Project
	matchParametersEnv.Environment = matchInfoDef.Environment

	return matchParametersEnv, errors

}

func getMapKeys(theMap map[string]bool) []string {
	keys := make([]string, len(theMap))

	i := 0
	for k := range theMap {
		keys[i] = k
		i++
	}

	return keys
}

func parseMatchFile(context *matchLoaderContext) (MatchFileDefinition, error) {

	data, err := afero.ReadFile(context.fs, context.matchFilePath)

	if err != nil {
		return MatchFileDefinition{}, err
	}

	if len(data) == 0 {
		return MatchFileDefinition{}, fmt.Errorf("file `%s` is empty", context.matchFilePath)
	}

	var result MatchFileDefinition

	err = yaml.UnmarshalStrict(data, &result)

	if err != nil {
		return MatchFileDefinition{}, err
	}

	return result, nil
}
