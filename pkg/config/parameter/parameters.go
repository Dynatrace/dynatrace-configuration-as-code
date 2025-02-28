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

package parameter

import (
	"fmt"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/strings"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/errors"
)

// Properties defines a map representing resolved parameters
type Properties map[string]interface{}

// PropertyResolver is used in parameter resolution to fetch the values of already deployed configs
type PropertyResolver interface {
	GetResolvedProperty(coordinate coordinate.Coordinate, propertyName string) (any, bool)
}

// ResolveContext used to give some more information on the resolving phase
type ResolveContext struct {
	PropertyResolver PropertyResolver

	// coordinates of the current config
	ConfigCoordinate coordinate.Coordinate

	// group of the current config
	Group string

	// environment of the current config
	Environment string

	// name of the parameter to resolve
	ParameterName string

	// resolved values of the current config
	ResolvedParameterValues Properties
}

type Parameter interface {
	// GetType returns the type of the parameter.
	GetType() string

	// GetReferences returns a slice of all other parameters this parameter references.
	// this is need in the sorting phase to first deploy references, so that
	// they an be resolved during deployment phase.
	GetReferences() []ParameterReference

	// ResolveValue resolves the value of this parameter. the context offers some more information
	// on the current deployment and resolving phase. if the value cannot be resolved,
	// an error should be returned.
	ResolveValue(context ResolveContext) (interface{}, error)
}

type NamedParameter struct {
	Name      string
	Parameter Parameter
}

// ParameterReference is used to identify a certain parameter in a config
type ParameterReference struct {
	Config   coordinate.Coordinate
	Property string
}

func (p ParameterReference) String() string {
	return fmt.Sprintf("%s:%s", p.Config, p.Property)
}

type ParameterParserContext struct {
	Folder        string
	Coordinate    coordinate.Coordinate
	Group         string
	Environment   string
	ParameterName string
	Fs            afero.Fs
	Value         map[string]interface {
	}
}

type ParameterParserError struct {
	// Location (coordinate) of the config the error happened in
	Location coordinate.Coordinate `json:"location"`
	// EnvironmentDetails of the environment the parsing of the parameter failed for
	EnvironmentDetails errors.EnvironmentDetails `json:"environmentDetails"`
	// ParameterName holds the name of the parameter triggering the error
	ParameterName string `json:"parameterName"`
	// Reason is a text describing what went wrong
	Reason string `json:"reason"`
}

func (p ParameterParserError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p ParameterParserError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p ParameterParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter: %s",
		p.ParameterName, p.Reason)
}

func NewParameterParserError(context ParameterParserContext, reason string) error {
	return ParameterParserError{
		Location:           context.Coordinate,
		EnvironmentDetails: errors.EnvironmentDetails{Group: context.Group, Environment: context.Environment},
		ParameterName:      context.ParameterName,
		Reason:             reason,
	}
}

type ParameterWriterError struct {
	// Location (coordinate) of the config the error happened in
	Location coordinate.Coordinate `json:"location"`
	// EnvironmentDetails of the environment the parsing of the parameter failed for
	EnvironmentDetails errors.EnvironmentDetails `json:"environmentDetails"`
	// name of the parameter triggering the error
	ParameterName string `json:"parameterName"`
	// text describing what went wrong
	Reason string `json:"reason"`
}

func (p ParameterWriterError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p ParameterWriterError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p ParameterWriterError) Error() string {
	return fmt.Sprintf("%s: cannot write parameter: %s",
		p.ParameterName, p.Reason)
}

func NewParameterWriterError(context ParameterWriterContext, reason string) error {
	return &ParameterWriterError{
		Location:           context.Coordinate,
		EnvironmentDetails: errors.EnvironmentDetails{Group: context.Group, Environment: context.Environment},
		ParameterName:      context.ParameterName,
		Reason:             reason,
	}
}

// ParameterResolveValueError is used to indicate that an error occurred during the resolving
// phase of a parameter.
type ParameterResolveValueError struct {
	// Location (coordinate) of the config.Config in which a parameter failed to be resolved
	Location coordinate.Coordinate `json:"location"`
	// EnvironmentDetails of the environment the resolving failed for
	EnvironmentDetails errors.EnvironmentDetails `json:"environmentDetails"`
	// ParameterName is the name of the parameter that failed to be resolved
	ParameterName string `json:"parameterName"`
	// Reason describing what went wrong
	Reason string `json:"reason"`
}

func (p ParameterResolveValueError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p ParameterResolveValueError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p ParameterResolveValueError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter: %s",
		p.ParameterName, p.Reason)
}

func NewParameterResolveValueError(context ResolveContext, reason string) ParameterResolveValueError {
	return ParameterResolveValueError{
		Location:           context.ConfigCoordinate,
		EnvironmentDetails: errors.EnvironmentDetails{Group: context.Group, Environment: context.Environment},
		ParameterName:      context.ParameterName,
		Reason:             reason,
	}
}

type ParameterWriterContext struct {
	// coordinates of the current config to parse
	Coordinate  coordinate.Coordinate
	Group       string
	Environment string
	// name of the current parameter to parse
	ParameterName string
	// current value to parse
	Parameter Parameter
}

// function loading a parameter from a given context
type ParameterParser func(ParameterParserContext) (Parameter, error)

// function used to transform a parameter to map structure, which can
// be serialized.
type ParameterWriter func(ParameterWriterContext) (map[string]interface{}, error)

// struct holding pointers to functions used to serialize
// and deserialize a parameter. this information is then used
// by the config loader.
type ParameterSerDe struct {
	Serializer   ParameterWriter
	Deserializer ParameterParser
}

// enforce that error types implement error interface
var (
	_ errors.DetailedConfigError = (*ParameterParserError)(nil)
	_ errors.DetailedConfigError = (*ParameterWriterError)(nil)
	_ errors.DetailedConfigError = (*ParameterResolveValueError)(nil)
)

func ToParameterReferences(params []interface{}, coord coordinate.Coordinate) (paramRefs []ParameterReference, err error) {
	for _, param := range params {
		switch param.(type) {
		case []interface{}, map[interface{}]interface{}:
			return nil, fmt.Errorf("error creating parameter reference: %v is not a string", param)
		}

		paramRefs = append(paramRefs, ParameterReference{
			Config:   coord,
			Property: strings.ToString(param),
		})
	}
	return paramRefs, nil
}
