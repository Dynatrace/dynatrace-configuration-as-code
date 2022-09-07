/**
 * @license
 * Copyright 2022 Dynatrace LLC
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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"regexp"
)

// RegEx matching environment variable template reference of the format '{{ .Env.NAME_OF_VAR }}'
// accepting any possible whitespace before and after the actual variable,
// and capturing the name of the actual env var (anything following .Env. up to the first whitespace or } character)
var envPattern = regexp.MustCompile(`{{\s*\.Env\.(.*?)\s*}}`)

// IsEnvVariable checks if a given string conforms to how monaco expects an environment variable reference looks in a
// template ( '{{ .Env.NAME_OF_VAR }}' )
func IsEnvVariable(s string) bool {
	return envPattern.MatchString(s)
}

// TrimToEnvVariableName takes an environment variable reference of the format '{{ .Env.NAME_OF_VAR }}' and trims it down
// to just the name of the environment variable ('NAME_OF_VAR').
// If an input does not conform to the expected format of an environment variable, this will return the input back as is.
func TrimToEnvVariableName(envReference string) string {
	if !IsEnvVariable(envReference) {
		return envReference
	}
	matches := envPattern.FindStringSubmatch(envReference)
	if len(matches) != 2 {
		log.Error("RegEx pattern matching returned %d matches, rather than expected 2 - full match & variable name capture group", len(matches))
		return envReference
	}
	return matches[1] //first and only capture group content returned on index 1, with full match on index 0
}
