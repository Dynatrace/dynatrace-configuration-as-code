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
	_ ConfigError = (*DetailedConfigWriterError)(nil)
)

// ConfigWriterError is an error that occurred while trying to write a config.Config to a file
type ConfigWriterError struct {
	// Path of the file that failed to be written
	Path string `json:"path"`
	// Err is the underlying error that occurred
	Err error `json:"error" jsonschema:"type=object"`
}

func (e ConfigWriterError) Unwrap() error {
	return e.Err
}

func (e ConfigWriterError) Error() string {
	return fmt.Sprintf("failed to write config to file %q: %s", e.Path, e.Err)
}

// DetailedConfigWriterError is an error that occurred while trying to write a config.Config to a file
type DetailedConfigWriterError struct {
	// Location (coordinate) of the config.Config that failed to be written to file
	Location coordinate.Coordinate `json:"location"`
	// Path of the file that failed to be written
	Path string `json:"path"`
	// Err is the underlying error that occurred
	Err error `json:"error" jsonschema:"type=object"`
}

func (e DetailedConfigWriterError) Unwrap() error {
	return e.Err
}

func (e DetailedConfigWriterError) Error() string {
	return fmt.Sprintf("failed to write config %s to file %q: %s", e.Location, e.Path, e.Err)
}

func (e DetailedConfigWriterError) Coordinates() coordinate.Coordinate {
	return e.Location
}
