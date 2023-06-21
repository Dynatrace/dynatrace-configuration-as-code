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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/coordinate"
)

var (
	_ ConfigError = (*DetailedConfigWriterError)(nil)
)

type ConfigWriterError struct {
	Path string
	Err  error
}

func (e ConfigWriterError) Unwrap() error {
	return e.Err
}

func (e ConfigWriterError) Error() string {
	return fmt.Sprintf("failed to write config to file %q: %s", e.Path, e.Err)
}

type DetailedConfigWriterError struct {
	Path     string
	Location coordinate.Coordinate
	Err      error
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
