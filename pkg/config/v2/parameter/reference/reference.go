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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/parameter"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util"
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

func (p *ReferenceParameter) GetType() string {
	return ReferenceParameterType
}

// UnresolvedReferenceError is indicating that the referenced parameter cannot be found and hence no value
// for the parameter been resolved.
type UnresolvedReferenceError struct {
	// define some shared error information
	parameter.ParameterResolveValueError

	// parameter which has not been resolved yet
	parameter.ParameterReference
}

func (e *UnresolvedReferenceError) Error() string {
	return fmt.Sprintf("%s: cannot resolve reference %s: %s",
		e.ParameterName, e.ParameterReference.ToString(), e.Reason)
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
		if val, found := context.ResolvedParameterValues[p.Property]; found {
			return val, nil
		}

		return nil, &UnresolvedReferenceError{
			ParameterResolveValueError: parameter.ParameterResolveValueError{
				Location: context.ConfigCoordinate,
				EnvironmentDetails: errors.EnvironmentDetails{
					Group:       context.Group,
					Environment: context.Environment,
				},
				ParameterName: context.ParameterName,
				Reason:        "property has not been resolved yet or does not exist",
			},
			ParameterReference: p.ParameterReference,
		}
	}

	if entity, found := context.ResolvedEntities[p.Config]; found {
		if val, found := entity.Properties[p.Property]; found {
			return val, nil
		}

		return nil, &UnresolvedReferenceError{
			ParameterResolveValueError: parameter.ParameterResolveValueError{
				Location: context.ConfigCoordinate,
				EnvironmentDetails: errors.EnvironmentDetails{
					Group:       context.Group,
					Environment: context.Environment,
				},
				ParameterName: context.ParameterName,
				Reason:        "property has not been resolved yet or does not exist",
			},
			ParameterReference: p.ParameterReference,
		}
	}

	return nil, &UnresolvedReferenceError{
		ParameterResolveValueError: parameter.ParameterResolveValueError{
			Location:      context.ConfigCoordinate,
			ParameterName: context.ParameterName,
			Reason:        "config has not been resolved yet or does not exist",
		},
		ParameterReference: p.ParameterReference,
	}
}

func writeReferenceParameter(context parameter.ParameterWriterContext) (map[string]interface{}, error) {
	refParam, ok := context.Parameter.(*ReferenceParameter)

	if !ok {
		return nil, &parameter.ParameterWriterError{
			Location:      context.Coordinate,
			ParameterName: context.ParameterName,
			Reason:        "unexpected type. parameter is not of type `ReferenceParameter`",
		}
	}

	result := make(map[string]interface{})
	sameProject := context.Coordinate.Project == refParam.Config.Project
	sameApi := context.Coordinate.Api == refParam.Config.Api
	sameConfig := context.Coordinate.Config == refParam.Config.Config

	if sameProject {
		result["project"] = refParam.Config.Project
	}

	if !sameProject || !sameApi {
		result["api"] = refParam.Config.Api
	}

	if !sameProject || !sameApi || !sameConfig {
		result["config"] = refParam.Config.Config
	}

	result["property"] = refParam.Property

	return result, nil
}

// parseReferenceParameter tries to parse a ReferenceParameter from a given context.
// it requires at least a `property` config value. All other values (project, api, config)
// will be filled in from the current context if missing.  it is not allowed to leave
// for example only `api` empty.
func parseReferenceParameter(context parameter.ParameterParserContext) (parameter.Parameter, error) {
	project := context.Coordinate.Project
	api := context.Coordinate.Api
	config := context.Coordinate.Config
	var property string
	projectSet := false
	apiSet := false
	configSet := false

	if val, ok := context.Value["project"]; ok {
		projectSet = true
		project = util.ToString(val)
	}

	if val, ok := context.Value["api"]; ok {
		apiSet = true
		api = util.ToString(val)
	}

	if val, ok := context.Value["config"]; ok {
		configSet = true
		config = util.ToString(val)
	}

	if val, ok := context.Value["property"]; ok {
		property = util.ToString(val)
	} else {
		return nil, &parameter.ParameterParserError{
			Location: context.Coordinate,
			EnvironmentDetails: errors.EnvironmentDetails{
				Group:       context.Group,
				Environment: context.Environment,
			},
			ParameterName: context.ParameterName,
			Reason:        "missing configuration `property`",
		}
	}

	// ensure that we do not have "holes" in the reference definition
	if projectSet && (!apiSet || !configSet) {
		return nil, &parameter.ParameterParserError{
			Location:      context.Coordinate,
			ParameterName: context.ParameterName,
			Reason:        "project is set, but either api or config isn't! please specify api and config",
		}
	}

	if apiSet && !configSet {
		return nil, &parameter.ParameterParserError{
			Location:      context.Coordinate,
			ParameterName: context.ParameterName,
			Reason:        "api is set, but config isn't! please specify config",
		}
	}

	return &ReferenceParameter{
		ParameterReference: parameter.ParameterReference{
			Config: coordinate.Coordinate{
				Project: project,
				Api:     api,
				Config:  config,
			},
			Property: property,
		},
	}, nil
}
