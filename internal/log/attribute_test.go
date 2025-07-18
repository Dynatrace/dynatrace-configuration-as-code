/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/coordinate"
)

func TestEnvironmentAttr(t *testing.T) {
	attr := EnvironmentAttr("env", "group")
	assert.Equal(t, "environment=[group=group name=env]", attr.String())
}

type customError struct {
}

func (customError) Error() string {
	return "custom error"
}

func TestErrorAttr(t *testing.T) {
	attr := ErrorAttr(customError{})

	assert.Equal(t, "error=[type=log.customError details=custom error]", attr.String())
}

func TestTypeAttr(t *testing.T) {
	attr := TypeAttr("my-type")

	assert.Equal(t, "type=my-type", attr.String())
}

func TestCoordinateAttr(t *testing.T) {
	attr := CoordinateAttr(coordinate.Coordinate{
		Project:  "p",
		Type:     "t",
		ConfigId: "c",
	})

	assert.Equal(t, "coordinate=p:t:c", attr.String())
}
