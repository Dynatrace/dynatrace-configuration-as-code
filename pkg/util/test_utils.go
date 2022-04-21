//go:build unit || integration
// +build unit integration

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package util

import (
	"os"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func SetEnv(t *testing.T, key string, value string) {
	err := os.Setenv(key, value)
	assert.NilError(t, err)
}

func UnsetEnv(t *testing.T, key string) {
	err := os.Unsetenv(key)
	assert.NilError(t, err)
}

func ReplaceName(line string, idChange func(string) string) string {

	if strings.Contains(line, "env-token-name:") {
		return line
	}

	if strings.Contains(line, "name:") {

		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "-") {
			trimmed = trimmed[1:]
			trimmed = strings.TrimSpace(trimmed)
		} else {
			return line
		}

		withoutPrefix := strings.TrimLeft(trimmed, "name:")
		name := strings.TrimSpace(withoutPrefix)

		if strings.HasPrefix(name, "\"") || strings.HasPrefix(name, "'") {
			name = name[1 : len(name)-1]
		}

		// Dependencies are not substituted
		isIdDependency := strings.HasSuffix(name, ".id")
		isNameDependency := strings.HasSuffix(name, ".name")

		if isIdDependency || isNameDependency {
			return line
		}

		replaced := strings.ReplaceAll(line, name, idChange(name))
		return replaced
	}
	return line
}
