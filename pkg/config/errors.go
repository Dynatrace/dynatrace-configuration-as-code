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

package config

import (
	"fmt"
	"strings"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	configErrors "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

var _ configErrors.DetailedConfigError = (*ParamsRefErr)(nil)

type ParamsRefErr struct {
	Location      coordinate.Coordinate        `json:"location"`
	ParameterName string                       `json:"parameterName"`
	Reference     parameter.ParameterReference `json:"parameterReference"`
	Reason        string                       `json:"reason"`
}

func newParamsRefErr(coord coordinate.Coordinate,
	param string, ref parameter.ParameterReference, reason string) ParamsRefErr {
	return ParamsRefErr{
		Location:      coord,
		ParameterName: param,
		Reference:     ref,
		Reason:        reason,
	}
}

func (e ParamsRefErr) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e ParamsRefErr) Error() string {
	return fmt.Sprintf("parameter `%s` cannot reference `%s`: %s",
		e.ParameterName, e.Reference, e.Reason)
}

var (
	_ error                            = (*CircularDependencyParameterSortError)(nil)
	_ configErrors.DetailedConfigError = (*CircularDependencyParameterSortError)(nil)
)

type CircularDependencyParameterSortError struct {
	Location      coordinate.Coordinate          `json:"location"`
	ParameterName string                         `json:"parameterName"`
	DependsOn     []parameter.ParameterReference `json:"dependsOn"`
}

func (e CircularDependencyParameterSortError) Coordinates() coordinate.Coordinate {
	return e.Location
}

func (e CircularDependencyParameterSortError) Error() string {
	return fmt.Sprintf("%s: circular dependency detected. check parameter dependencies: %s",
		e.ParameterName, joinParameterReferencesToString(e.DependsOn))
}

func joinParameterReferencesToString(refs []parameter.ParameterReference) string {
	switch len(refs) {
	case 0:
		return ""
	case 1:
		return refs[0].String()
	}

	result := strings.Builder{}

	for _, ref := range refs {
		result.WriteString(ref.String())
		result.WriteString(", ")
	}

	return result.String()
}
