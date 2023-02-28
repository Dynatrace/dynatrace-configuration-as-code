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

package v1environment

import (
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/template"
	"github.com/spf13/afero"
)

// LoadEnvironmentsWithoutTemplating loads environments from a yaml file without templating. No variable references will
// be replaced on loading.
func LoadEnvironmentsWithoutTemplating(environmentsFile string, fs afero.Fs) (environments map[string]*EnvironmentV1, errorList []error) {
	if environmentsFile == "" {
		errorList = append(errorList, errors.New("no environment file provided"))
		return environments, errorList
	}

	dat, err := afero.ReadFile(fs, environmentsFile)
	errutils.FailOnError(err, "Error while reading file")

	environmentMaps, err := template.UnmarshalYamlWithoutTemplating(string(dat), environmentsFile)
	errutils.FailOnError(err, "Error while converting file")

	environments, envErrs := newEnvironmentsV1(environmentMaps)

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
