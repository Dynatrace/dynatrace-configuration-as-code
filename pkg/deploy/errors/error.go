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

package errors

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"strings"
)

var (
	_ configErrors.DetailedConfigError = (*ConfigDeployErr)(nil)
)

type ConfigDeployErr struct {
	Location           coordinate.Coordinate           `json:"location"`
	EnvironmentDetails configErrors.EnvironmentDetails `json:"environmentDetails"`
	Reason             string                          `json:"reason"`
	Err                error                           `json:"error"`
}

func NewConfigDeployErr(conf *config.Config, reason string) ConfigDeployErr {
	return ConfigDeployErr{
		Location: conf.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       conf.Group,
			Environment: conf.Environment,
		},
		Reason: reason,
	}
}

func (e ConfigDeployErr) WithError(err error) ConfigDeployErr {
	e.Err = err
	return e
}

func (e ConfigDeployErr) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e ConfigDeployErr) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e ConfigDeployErr) Unwrap() error {
	return e.Err
}

func (e ConfigDeployErr) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Reason
}

type EnvironmentDeploymentErrors map[string][]error

func (e EnvironmentDeploymentErrors) Error() string {
	b := strings.Builder{}
	for env, errs := range e {
		b.WriteString(fmt.Sprintf("%s deployment errors: %v", env, errs))
	}
	return b.String()
}

func (e EnvironmentDeploymentErrors) Append(env string, err ...error) EnvironmentDeploymentErrors {
	if _, exists := e[env]; !exists {
		e[env] = make([]error, 0)
	}
	e[env] = append(e[env], err...)
	return e
}
