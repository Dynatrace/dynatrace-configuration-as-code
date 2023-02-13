/**
 * @license
 * Copyright 2022 Dynatrace LLC
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

package util

import (
	"fmt"
	"path/filepath"

	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/manifest"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/spf13/afero"
)

type EnvCredentials struct {
	EnvUrl      string
	Token       string
	TokenEnvVar string
}

func GetEnvFromManifest(fs afero.Fs, manifestPath string, specificEnvironmentName string) (envCredentials EnvCredentials, err error) {

	var errs []error

	man, err := GetManifest(fs, manifestPath)
	if err != nil {
		return
	}

	env, found := man.Environments[specificEnvironmentName]
	if !found {
		err = fmt.Errorf("environment '%v' was not available in manifest '%v'", specificEnvironmentName, manifestPath)
		return
	}

	if len(errs) > 0 {
		err = fmt.Errorf("failed to load apis")
		return
	}

	envCredentials.EnvUrl, err = env.GetUrl()
	if err != nil {
		errs = append(errs, err)
	}

	envCredentials.Token, err = env.GetToken()
	if err != nil {
		errs = append(errs, err)
	}

	if envVarToken, ok := env.Token.(*manifest.EnvironmentVariableToken); ok {
		envCredentials.TokenEnvVar = envVarToken.EnvironmentVariableName
	} else {
		errs = append(errs, fmt.Errorf("env token not found"))
	}

	if len(errs) > 0 {
		err = util.PrintAndFormatErrors(errs, "failed to load manifest data")
	}

	return
}

func GetManifest(fs afero.Fs, manifestPath string) (manifest.Manifest, error) {
	man, errs := manifest.LoadManifest(&manifest.ManifestLoaderContext{
		Fs:           fs,
		ManifestPath: manifestPath,
	})

	if errs != nil {
		err := util.PrintAndFormatErrors(errs, "failed to load manifest '%v'", manifestPath)
		return manifest.Manifest{}, err
	}

	return man, nil
}

func GetFilePaths(fileName string) (string, string, error) {
	filePath := filepath.Clean(fileName)
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", "", err
	}

	fileWorkingDir := filepath.Dir(fileName)
	return fileWorkingDir, filePath, nil
}
