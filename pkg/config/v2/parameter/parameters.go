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

	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/coordinate"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/config/v2/errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
)

// Properties defines a map representing resolved parameters
type Properties map[string]interface{}

// ResolvedEntities defines a map representing resolved configs. this includes the
// api `ID` of a config.
type ResolvedEntities map[coordinate.Coordinate]ResolvedEntity

// TODO move to better package
// ResolvedEntity struct representing an already deployed entity
type ResolvedEntity struct {
	// EntityName is the name returned by the Dynatrace api. In theory should be the
	// same as the `name` property defined in the configuration, but
	// can differ.
	EntityName string

	// coordinate of the config this entity represents
	Coordinate coordinate.Coordinate

	// Properties defines a map of all already resolved parameters
	Properties Properties

	// Skip flag indicating that this entity was skipped
	// if a entity is skipped, there will be no properties
	Skip bool
}

// ResolveContext used to give some more information on the resolving phase
type ResolveContext struct {
	// map of already resolved (and deployed) configs
	ResolvedEntities ResolvedEntities

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

	Client rest.DynatraceClient

	// dry run indicates that no persistent operations should be made
	DryRun bool
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

// ParameterReference is used to identify a certain parameter in a config
type ParameterReference struct {
	Config   coordinate.Coordinate
	Property string
}

func (p *ParameterReference) ToString() string {
	return fmt.Sprintf("%s:%s", p.Config.ToString(), p.Property)
}

type ParameterParserContext struct {
	// coordinates of the current config to parse
	Coordinate  coordinate.Coordinate
	Group       string
	Environment string
	// name of the current parameter to parse
	ParameterName string
	// current value to parse
	Value map[string]interface{}
}

type ParameterParserError struct {
	// Location of the config the error happened in
	Location           coordinate.Coordinate
	EnvironmentDetails errors.EnvironmentDetails
	// ParameterName holds the name of the parameter triggering the error
	ParameterName string
	// Reason is a text describing what went wrong
	Reason string
}

func (p *ParameterParserError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p *ParameterParserError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p *ParameterParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter: %s",
		p.ParameterName, p.Reason)
}

type ParameterWriterError struct {
	// config the error happened in
	Location           coordinate.Coordinate
	EnvironmentDetails errors.EnvironmentDetails
	// name of the parameter triggering the error
	ParameterName string
	// text describing what went wrong
	Reason string
}

func (p *ParameterWriterError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p *ParameterWriterError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p *ParameterWriterError) Error() string {
	return fmt.Sprintf("%s: cannot write parameter: %s",
		p.ParameterName, p.Reason)
}

// ParameterResolveValueError is used to indicate that an error occurred during the resolving
// phase of a parameter.
type ParameterResolveValueError struct {
	Location           coordinate.Coordinate
	EnvironmentDetails errors.EnvironmentDetails
	ParameterName      string
	Reason             string
}

func (p *ParameterResolveValueError) Coordinates() coordinate.Coordinate {
	return p.Location
}

func (p *ParameterResolveValueError) LocationDetails() errors.EnvironmentDetails {
	return p.EnvironmentDetails
}

func (p *ParameterResolveValueError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter: %s",
		p.ParameterName, p.Reason)
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
