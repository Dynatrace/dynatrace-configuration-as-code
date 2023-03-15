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

package regex

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"regexp"
)

// EnvVariableRegexPattern matching environment variable template reference of the format '{{ .Env.NAME_OF_VAR }}'
// accepting any possible whitespace before and after the actual variable,
// and capturing the name of the actual env var (anything following .Env. up to the first whitespace or } character)
var EnvVariableRegexPattern = regexp.MustCompile(`{{ *\.Env\.([A-Za-z0-9_-]*) *}}`)

// IsEnvVariable checks if a given string conforms to how monaco expects an environment variable reference looks in a
// template ( '{{ .Env.NAME_OF_VAR }}' )
func IsEnvVariable(s string) bool {
	return EnvVariableRegexPattern.MatchString(s)
}

// TrimToEnvVariableName takes an environment variable reference of the format '{{ .Env.NAME_OF_VAR }}' and trims it down
// to just the name of the environment variable ('NAME_OF_VAR').
// If an input does not conform to the expected format of an environment variable, this will return the input back as is.
func TrimToEnvVariableName(envReference string) string {
	if !IsEnvVariable(envReference) {
		return envReference
	}
	matches := EnvVariableRegexPattern.FindStringSubmatch(envReference)
	if len(matches) != 2 {
		log.Error("RegEx pattern matching returned %d matches, rather than expected 2 - full match & variable name capture group", len(matches))
		return envReference
	}
	return matches[1] // first and only capture group content returned on index 1, with full match on index 0
}

// pattern matching strings of the format '"value", "value", ...' which are sometimes used to set lists into JSON templates
// these must generally not have their quotes escaped as their JSON template is usually not valid with these values
var listDefinitionRegex = regexp.MustCompile(`(?:\s*".*?"\s*,\s*".*?"\s*,?)+`)

// IsListDefinition checks if a given string conforms to a pattern of the format '"value", "value", ...'
// which are sometimes used to set lists into JSON templates
func IsListDefinition(s string) bool {
	return listDefinitionRegex.MatchString(s)
}

// simple regex matching anything that is text between double quotes
// Example: "some text@@(*$*&!(#"
var simpleValueRegex = regexp.MustCompile(`\s*".+?"\s*`)

// IsSimpleValueDefinition checks if a given string is any text between double quotes
func IsSimpleValueDefinition(s string) bool {
	return simpleValueRegex.MatchString(s)
}

// ListVariableRegexPattern matching list references in a Template,
// capturing the variable name as well as the enclosing square bracket block
// Sample format: "listKey": [ {{.list_variable}} ], captures: "[ {{.list_variable}} ]" and "list_variable"
var ListVariableRegexPattern = regexp.MustCompile(`"[\w\s]+"\s*:\s*(\[\s*\{\{\s*\.([\w]+)\s*}}\s*])`)

func MatchListVariable(s string) (fullMatch string, listMatch string, variableName string, err error) {
	match := ListVariableRegexPattern.FindStringSubmatch(s)

	if len(match) != 3 {
		return "", "", "", fmt.Errorf("cannot parse list variable: `%s` seems to be invalid", s)
	}

	return match[0], match[1], match[2], nil
}
