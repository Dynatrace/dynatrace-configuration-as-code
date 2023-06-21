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

package deploy

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

var (
	_ configErrors.DetailedConfigError = (*configDeployErr)(nil)
	_ configErrors.DetailedConfigError = (*paramsRefErr)(nil)
)

type paramsRefErr struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	Parameter          string
	Reference          parameter.ParameterReference
	Reason             string
}

func newParamsRefErr(coord coordinate.Coordinate, group string, env string,
	param string, ref parameter.ParameterReference, reason string) paramsRefErr {
	return paramsRefErr{
		Config: coord,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       group,
			Environment: env,
		},
		Parameter: param,
		Reference: ref,
		Reason:    reason,
	}
}

func (e paramsRefErr) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e paramsRefErr) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e paramsRefErr) Error() string {
	return fmt.Sprintf("parameter `%s` cannot reference `%s`: %s",
		e.Parameter, e.Reference, e.Reason)
}

type configDeployErr struct {
	Config             coordinate.Coordinate
	EnvironmentDetails configErrors.EnvironmentDetails
	Reason             string
	Err                error
}

func newConfigDeployErr(conf *config.Config, reason string) configDeployErr {
	return configDeployErr{
		Config: conf.Coordinate,
		EnvironmentDetails: configErrors.EnvironmentDetails{
			Group:       conf.Group,
			Environment: conf.Environment,
		},
		Reason: reason,
	}
}

func (e configDeployErr) withError(err error) configDeployErr {
	e.Err = err
	return e
}

func (e configDeployErr) Coordinates() coordinate.Coordinate {
	return e.Config
}

func (e configDeployErr) LocationDetails() configErrors.EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e configDeployErr) Unwrap() error {
	return e.Err
}

func (e configDeployErr) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Reason
}
