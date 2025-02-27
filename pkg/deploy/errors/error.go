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
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
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

func NewFromErr(conf *config.Config, err error) ConfigDeployErr {
	return ConfigDeployErr{
		Location: conf.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       conf.Group,
			Environment: conf.Environment,
		},
		Err: err,
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
	b.WriteString(fmt.Sprintf("Errors encountered for %d environment(s):", len(e)))
	for env, errs := range e {
		if len(errs) == 1 {
			b.WriteString(fmt.Sprintf("\n\t%q: %v", env, errs[0]))
		} else {
			b.WriteString(fmt.Sprintf("\n\t%q:", env))
			for _, err := range errs {
				b.WriteString(fmt.Sprintf("\n\t\t- %v", err))
			}
		}
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

// DeploymentErrors is an error returned if any deployment errors occured. It carries a count of how many errors happened
// during deployment, but no details on those errors. The specific errors that have happened during deployment are handled
// by logging them, and never returned out of DeployConfigGraph.
type DeploymentErrors struct {
	// ErrorCount tells how many errors occurred during a deployment
	ErrorCount int
}

func (d DeploymentErrors) Error() string {
	return fmt.Sprintf("%d deployment errors occurred", d.ErrorCount)
}
