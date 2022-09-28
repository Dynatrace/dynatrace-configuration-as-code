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
	"encoding/json"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"reflect"
	"regexp"
	"strings"
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

// EscapeSpecialCharacters walks recursively though the map and escapes all special characters that can't just be written to the
// json template. characters that will be escaped: newlines (\n), double quotes (\")
// Note: this is in use in both v1 and v2 config templating
func EscapeSpecialCharacters(properties map[string]interface{}) (map[string]interface{}, error) {

	escapedProperties := make(map[string]interface{}, len(properties))

	for key, value := range properties {

		switch field := value.(type) {
		case string:
			escaped, err := escapeCharacters(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		case map[string]string:
			escaped, err := escapeCharactersForStringMap(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		case map[string]interface{}:
			escaped, err := EscapeSpecialCharacters(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		default:
			log.Debug("Unknown value type %v in property %v.", reflect.TypeOf(value), key)
		}
	}

	return escapedProperties, nil
}

func escapeCharactersForStringMap(properties map[string]string) (map[string]string, error) {
	escapedProperties := make(map[string]string, len(properties))

	for key, value := range properties {
		escaped, err := escapeCharacters(value)
		if err != nil {
			return nil, err
		}
		escapedProperties[key] = escaped
	}

	return escapedProperties, nil
}

func escapeCharacters(rawString string) (string, error) {
	if isListDefinition(rawString) {
		return rawString, nil
	}
	return escapeNewlines(rawString), nil
}

// Due to APM-387662 this is currently NOT used
//
// escapeCharactersForJson ensures a string can be placed into a json by just marshalling it to json.
// This will escape anything that needs to be escaped - but explicitly excludes strings that are of string list format.
// Such list strings can be used to place several values into a json list and their double-quotes are needed to render
// valid json and must not be escaped. As a caveat this means any other characters aren't escaped either for lists.
// As marshalling additionally places quotes around the output these first and last characters are cut off before returning.
func escapeCharactersForJson(rawString string) (string, error) {
	if isListDefinition(rawString) {
		return rawString, nil
	}

	b, err := json.Marshal(rawString)
	if err != nil {
		// errors should never occur for marshalling a string value - better safe than sorry if implementation details change
		return "", err
	}
	s := string(b)
	s = s[1 : len(s)-1] //marshalling places quotes around the json string which we don't want
	return s, nil
}

// escapeNewlines only escapes newline characters in an input string by replacing all occurrences with a raw \n
func escapeNewlines(rawString string) string {
	return strings.ReplaceAll(rawString, "\n", `\n`)
}

// pattern matching strings of the format '"value", "value", ...' which are sometimes used to set lists into JSON templates
// these must generally not have their quotes escaped as their JSON template is usually not valid with these values
var listPattern = regexp.MustCompile(`(?:\s*".*?"\s*,\s*".*?"\s*,?)+`)

func isListDefinition(s string) bool {
	return listPattern.MatchString(s)
}
