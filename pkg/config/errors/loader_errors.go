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

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

var (
	_ ConfigError         = (*DefinitionParserError)(nil)
	_ DetailedConfigError = (*DetailedDefinitionParserError)(nil)
	_ DetailedConfigError = (*ParameterDefinitionParserError)(nil)
)

// ConfigLoaderError contains details about an error that occurred while loading a config.Config
type ConfigLoaderError struct {
	// Path of the config.yaml that could not be loaded
	Path string `json:"path"`
	// Err is the underlying error that occurred while loading
	Err error `json:"error" jsonschema:"type=object"`
}

func (e ConfigLoaderError) Unwrap() error {
	return e.Err
}

func (e ConfigLoaderError) Error() string {
	return fmt.Sprintf("failed to load config from file %q: %s", e.Path, e.Err)
}

// DefinitionParserError contains details about errors when parsing a YAML definition of a config
type DefinitionParserError struct {
	// Location (coordinate) of the configuration that could not be parsed
	Location coordinate.Coordinate `json:"location"`
	// Path of the config.yaml that could not be parsed
	Path string `json:"path"`
	// Reason describing why parsing failed
	Reason string `json:"reason"`
}

// DetailedDefinitionParserError is a DefinitionParserError, enriched with information for which environment loading failed
type DetailedDefinitionParserError struct {
	DefinitionParserError
	// EnvironmentDetails of the environment the parsing of a configuration failed for
	EnvironmentDetails EnvironmentDetails `json:"environmentDetails"`
}

func (e DetailedDefinitionParserError) LocationDetails() EnvironmentDetails {
	return e.EnvironmentDetails
}

func (e DetailedDefinitionParserError) Environment() string {
	return e.EnvironmentDetails.Environment
}

func (e DefinitionParserError) Coordinates() coordinate.Coordinate {
	return e.Location
}

// ParameterDefinitionParserError is a DetailedDefinitionParserError for a specific parameter that failed to laod
type ParameterDefinitionParserError struct {
	DetailedDefinitionParserError
	// ParameterName of the YAML parameter that failed to be parsed
	ParameterName string `json:"parameterName"`
}

func (e ParameterDefinitionParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter definition in `%s`: %s",
		e.ParameterName, e.Path, e.Reason)
}

func (e DefinitionParserError) Error() string {
	return fmt.Sprintf("cannot parse definition in `%s`: %s",
		e.Path, e.Reason)
}

type UnknownEnvironmentError struct {
	EnvironmentName string
}

func (e UnknownEnvironmentError) Error() string {
	return fmt.Sprintf("unknown environment '%s'", e.EnvironmentName)
}

type UnknownEnvironmentGroupError struct {
	GroupName string
}

func (e UnknownEnvironmentGroupError) Error() string {
	return fmt.Sprintf("unknown environment group '%s'", e.GroupName)
}
