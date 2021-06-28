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

package environment

import (
	"fmt"

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/envvars"
)

// EnvironmentVariableParameterType specifies the type of the parameter used in config files
const EnvironmentVariableParameterType = "environment"

var EnvironmentVariableParameterSerde = parameter.ParameterSerDe{
	Serializer:   writeEnvironmentValueParameter,
	Deserializer: parseEnvironmentValueParameter,
}

// EnvironmentVariableParameter defines a parameter which can load an value from the
// environment variables. there is even the possibility to define a default value,
// if the one from the environment is missing.
type EnvironmentVariableParameter struct {
	// name of the referenced environment variable
	Name string

	// default value used if environment variable specified by `name` cannot be found.
	// note: this value is only used, if the `HasDefaultValue` flag is set to true.
	DefaultValue string

	// flag indicating that a default value has been set. this is needed, as
	// we cannot distinguish an empty string from an not set value.
	HasDefaultValue bool
}

// this forces the compiler to check if EnvironmentVariableParameter is of type Parameter
var _ parameter.Parameter = (*EnvironmentVariableParameter)(nil)

func (p *EnvironmentVariableParameter) GetType() string {
	return EnvironmentVariableParameterType
}

func (p *EnvironmentVariableParameter) GetReferences() []parameter.ParameterReference {
	// environment variable parameters cannot have references
	return []parameter.ParameterReference{}
}

func (p *EnvironmentVariableParameter) ResolveValue(context parameter.ResolveContext) (interface{}, error) {
	if val, found := envvars.Lookup(p.Name); found {
		return val, nil
	}

	if p.HasDefaultValue {
		return p.DefaultValue, nil
	}

	return nil, &parameter.ParameterResolveValueError{
		Location: context.ConfigCoordinate,
		EnvironmentDetails: errors.EnvironmentDetails{
			Group:       context.Group,
			Environment: context.Environment,
		},
		ParameterName: context.ParameterName,
		Reason:        fmt.Sprintf("environment variable `%s` not set", p.Name),
	}
}

// parseEnvironmentValueParameter parses an EnvironmentVariableParameter from a given context.
// it requires a `name` field to be set. `default` is an optional field.
func parseEnvironmentValueParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	if name, ok := context.Value["name"]; ok {
		defaultValue := ""
		hasDefault := false

		if val, ok := context.Value["default"]; ok {
			defaultValue = util.ToString(val)
			hasDefault = true
		}

		return &EnvironmentVariableParameter{
			Name:            util.ToString(name),
			DefaultValue:    defaultValue,
			HasDefaultValue: hasDefault,
		}, nil

	}
	return nil, &parameter.ParameterParserError{
		Location: context.Coordinate,
		EnvironmentDetails: errors.EnvironmentDetails{
			Group:       context.Group,
			Environment: context.Environment,
		},
		ParameterName: context.ParameterName,
		Reason:        "missing property `name`",
	}
}

func writeEnvironmentValueParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	envParam, ok := context.Parameter.(*EnvironmentVariableParameter)

	if !ok {
		return nil, &parameter.ParameterWriterError{
			Location: context.Coordinate,
			EnvironmentDetails: errors.EnvironmentDetails{
				Group:       context.Group,
				Environment: context.Environment,
			},
			ParameterName: context.ParameterName,
			Reason:        "unexpected type. parameter is not of type `EnvironmentVariableParameter`",
		}
	}

	result := make(map[string]interface{})

	if envParam.HasDefaultValue {
		result["default"] = envParam.DefaultValue
	}

	result["name"] = envParam.Name

	return result, nil
}
