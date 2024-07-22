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

package reference

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/entities"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/parameter"
)

// ReferenceParameterType specifies the type of the parameter used in config files
const ReferenceParameterType = "reference"

var ReferenceParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeReferenceParameter,
	Deserializer: parseReferenceParameter,
}

// ReferenceParameter is a parameter which evaluates to the value of the referenced parameter.
type ReferenceParameter struct {
	parameter.ParameterReference
}

func New(project string, configType string, config string, property string) *ReferenceParameter {
	coord := coordinate.Coordinate{
		Project:  project,
		Type:     configType,
		ConfigId: config,
	}

	return &ReferenceParameter{
		parameter.ParameterReference{Config: coord, Property: property},
	}
}

func NewWithCoordinate(coordinate coordinate.Coordinate, property string) *ReferenceParameter {
	return &ReferenceParameter{
		parameter.ParameterReference{Config: coordinate, Property: property},
	}
}

func (p *ReferenceParameter) GetType() string {
	return ReferenceParameterType
}

func (p *ReferenceParameter) Equal(r *ReferenceParameter) bool {
	return p.ParameterReference == r.ParameterReference
}

// UnresolvedReferenceError is indicating that the referenced parameter cannot be found and hence no value
// for the parameter been resolved.
type UnresolvedReferenceError struct {
	// define some shared error information
	parameter.ParameterResolveValueError

	// parameter which has not been resolved yet
	parameter.ParameterReference
}

func (e UnresolvedReferenceError) Error() string {
	return fmt.Sprintf("%s: cannot resolve reference %s: %s",
		e.ParameterName, e.ParameterReference, e.Reason)
}

var (
	_ errors.DetailedConfigError = (*UnresolvedReferenceError)(nil)
	// this forces the compiler to check if ReferenceParameter is of type Parameter
	_ parameter.Parameter = (*ReferenceParameter)(nil)
)

func (p *ReferenceParameter) GetReferences() []parameter.ParameterReference {
	return []parameter.ParameterReference{p.ParameterReference}
}

// ResolveValue tries to find the reference in the already resolved entities. if the referenced entity
// cannot be found (maybe it is missing, or has not been resolved yet), an error is returned.
func (p *ReferenceParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	// in case we are referencing a parameter in the same config, we do not have to check
	// the resolved entities
	if context.ConfigCoordinate.Match(p.Config) {
		m := make(map[interface{}]any)
		for k, v := range context.ResolvedParameterValues {
			m[k] = v
		}
		if val, found := entities.ResolvePropValue(p.Property, m); found {
			return val, nil
		}
		return nil, newUnresolvedReferenceError(context, p.ParameterReference, "property has not been resolved yet or does not exist")
	}

	if context.PropertyResolver == nil {
		return nil, newUnresolvedReferenceError(context, p.ParameterReference, "no PropertyResolver is defined")
	}

	if val, found := context.PropertyResolver.GetResolvedProperty(p.Config, p.Property); found {
		return val, nil
	}

	return nil, newUnresolvedReferenceError(context, p.ParameterReference, "config has not been resolved yet or does not exist")
}

const projectField = "project"
const typeField = "configType"
const idField = "configId"
const propertyField = "property"

func writeReferenceParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	refParam, ok := context.Parameter.(*ReferenceParameter)

	if !ok {
		return nil, parameter.NewParameterWriterError(context, "unexpected type. parameter is not of type `ReferenceParameter`")
	}

	result := make(map[string]interface{})
	sameProject := context.Coordinate.Project == refParam.Config.Project
	sameType := context.Coordinate.Type == refParam.Config.Type
	sameConfig := context.Coordinate.ConfigId == refParam.Config.ConfigId

	if !sameProject {
		result[projectField] = refParam.Config.Project
	}

	if !sameProject || !sameType {
		result[typeField] = refParam.Config.Type
	}

	if !sameProject || !sameType || !sameConfig {
		result[idField] = refParam.Config.ConfigId
	}

	result[propertyField] = refParam.Property

	return result, nil
}

// parseReferenceParameter tries to parse a ReferenceParameter from a given context.
// it requires at least a `property` config value. All other values (project, type, config)
// will be filled in from the current context if missing.  it is not allowed to leave
// for example only `type` empty.
func parseReferenceParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	project := context.Coordinate.Project
	configType := context.Coordinate.Type
	config := context.Coordinate.ConfigId
	var property string
	projectSet := false
	typeSet := false
	configSet := false

	if val, ok := context.Value[projectField]; ok {
		projectSet = true
		project = strings.ToString(val)
	}

	if val, ok := context.Value[typeField]; ok {
		typeSet = true
		configType = strings.ToString(val)
	}

	if val, ok := context.Value[idField]; ok {
		configSet = true
		config = strings.ToString(val)
	}

	if val, ok := context.Value[propertyField]; ok {
		property = strings.ToString(val)
	} else {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("missing `%s` - please specifiy which %s should be referenced", propertyField, propertyField))
	}

	// ensure that we do not have "holes" in the reference definition
	if projectSet && (!typeSet || !configSet) {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("`%s` is set, but either `%s` or `%s` isn't! please specify `%s` and `%s`", projectField, typeField, idField, typeField, idField))
	}

	if typeSet && !configSet {
		return nil, parameter.NewParameterParserError(context, fmt.Sprintf("`%s` is set, but `%s` isn't! please specify `%s`", typeField, idField, idField))
	}

	return New(project, configType, config, property), nil
}

func newUnresolvedReferenceError(context parameter.ResolveContext, reference parameter.ParameterReference, reason string) error {
	return &UnresolvedReferenceError{
		ParameterResolveValueError: parameter.NewParameterResolveValueError(context, reason),
		ParameterReference:         reference,
	}
}
