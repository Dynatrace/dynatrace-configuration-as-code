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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/config/v2/coordinate"
)

var (
	_ ConfigError         = (*DefinitionParserError)(nil)
	_ DetailedConfigError = (*DetailedDefinitionParserError)(nil)
	_ DetailedConfigError = (*ParameterDefinitionParserError)(nil)
)

type ConfigLoaderError struct {
	Path string
	Err  error
}

func (e ConfigLoaderError) Unwrap() error {
	return e.Err
}

func (e ConfigLoaderError) Error() string {
	return fmt.Sprintf("failed to load config from file %q: %s", e.Path, e.Err)
}

type DefinitionParserError struct {
	Location coordinate.Coordinate
	Path     string
	Reason   string
}

type DetailedDefinitionParserError struct {
	DefinitionParserError
	EnvironmentDetails EnvironmentDetails
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

type ParameterDefinitionParserError struct {
	DetailedDefinitionParserError
	ParameterName string
}

func (e ParameterDefinitionParserError) Error() string {
	return fmt.Sprintf("%s: cannot parse parameter definition in `%s`: %s",
		e.ParameterName, e.Path, e.Reason)
}

func (e DefinitionParserError) Error() string {
	return fmt.Sprintf("cannot parse definition in `%s`: %s",
		e.Path, e.Reason)
}
