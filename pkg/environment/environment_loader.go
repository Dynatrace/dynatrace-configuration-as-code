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

package environment

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util"
	"github.com/spf13/afero"
)

// LoadEnvironmentList loads environments from a templated yaml file - any references to environment variables will be
// replaced with actual values. This will fail if an environment variable is referenced but not set.
// If a specificEnvironment is provided only this environment will be returned, if the requested environment does not exist,
// an error is returned.
func LoadEnvironmentList(specificEnvironment string, environmentsFile string, fs afero.Fs) (environments map[string]Environment, errorList []error) {

	if environmentsFile == "" {
		errorList = append(errorList, errors.New("no environment file provided"))
		return environments, errorList
	}

	environmentsFromFile, errorList := readEnvironments(environmentsFile, fs)

	if len(environmentsFromFile) == 0 {
		errorList = append(errorList, fmt.Errorf("no environments loaded from file %s", environmentsFile))
		return environments, errorList
	}

	if specificEnvironment != "" {
		if environmentsFromFile[specificEnvironment] == nil {
			errorList = append(errorList, fmt.Errorf("environment %s not found in file %s", specificEnvironment, environmentsFile))
			return environments, errorList
		}

		environments = make(map[string]Environment)
		environments[specificEnvironment] = environmentsFromFile[specificEnvironment]
	} else {
		environments = environmentsFromFile
	}

	return environments, errorList
}

// LoadEnvironmentsWithoutTemplating loads environments from a yaml file without templating. No variable references will
// be replaced on loading.
func LoadEnvironmentsWithoutTemplating(environmentsFile string, fs afero.Fs) (environments map[string]Environment, errorList []error) {
	if environmentsFile == "" {
		errorList = append(errorList, errors.New("no environment file provided"))
		return environments, errorList
	}

	dat, err := afero.ReadFile(fs, environmentsFile)
	util.FailOnError(err, "Error while reading file")

	environmentMaps, err := util.UnmarshalYamlWithoutTemplating(string(dat), environmentsFile)
	util.FailOnError(err, "Error while converting file")

	environments, envErrs := NewEnvironments(environmentMaps)

	if len(envErrs) > 0 {
		errorList = append(errorList, envErrs...)
		return environments, errorList
	}

	if len(environments) == 0 {
		errorList = append(errorList, fmt.Errorf("no environments loaded from file %s", environmentsFile))
		return environments, errorList
	}

	return environments, errorList
}

// readEnvironments reads the yaml file for the environments and returns the parsed environments
func readEnvironments(file string, fs afero.Fs) (map[string]Environment, []error) {

	dat, err := afero.ReadFile(fs, file)
	util.FailOnError(err, "Error while reading file")

	environmentMaps, err := util.UnmarshalYaml(string(dat), file)
	util.FailOnError(err, "Error while converting file")

	return NewEnvironments(environmentMaps)
}
