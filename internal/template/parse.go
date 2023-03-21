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

package template

import (
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/files"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// UnmarshalYamlFunc is a function that will Umarshal a yaml string of a config or environment definition into a map.
type UnmarshalYamlFunc func(text string, filename string) (map[string]map[string]string, error)

// UnmarshalYaml takes the contents of a yaml file and converts it to a map[string]map[string]string.
// The file be templated, with any references replaced - or resulting in an error if no value is available.
//
// The yaml file should have the following format:
//
// some-name-1:
//   - list-key-1: "list-entry-1"
//   - list-key-2: "list-entry-2"
//
// some-name-2:
//   - list-key-1: "list-entry-1"
func UnmarshalYaml(text string, fileName string) (map[string]map[string]string, error) {

	template, err := NewTemplateFromString(fileName, text)
	if err != nil {
		return make(map[string]map[string]string), err
	}

	text, err = template.ExecuteTemplate(make(map[string]string))
	if err != nil {
		return make(map[string]map[string]string), err
	}

	m := make(map[string]interface{})

	err = yaml.Unmarshal([]byte(text), &m)
	errutils.FailOnError(err, "Failed to unmarshal yaml\n"+text+"\nerror:")

	typed, err := convert(m)
	errutils.FailOnError(err, "YAML file "+fileName+" could not be parsed")

	return typed, nil
}

// UnmarshalYamlWithoutTemplating takes the contents of a yaml file and converts it to a map[string]map[string]string.
// If references should be replaced (which you generally want) use UnmarshalYaml instead.
func UnmarshalYamlWithoutTemplating(text string, fileName string) (map[string]map[string]string, error) {
	m := make(map[string]interface{})

	text = ensureAnyTemplateStringsAreInQuotes(text)

	err := yaml.Unmarshal([]byte(text), &m)
	errutils.FailOnError(err, "Failed to unmarshal yaml\n"+text+"\nerror:")

	typed, err := convert(m)
	errutils.FailOnError(err, "YAML file "+fileName+" could not be parsed")

	return typed, nil
}

// nonQuotedVariableRegex matches a limited edge case of variable definitons as yaml values defined without quotes
// e.g. - value: {{ .Env.MyValue }} is sometimes used in monaco configurations. Templating will replace this with the
// actual values in quotes, but without templating this will produce invalid yaml. This matches on a value (something
// after a colon) that does not start with a double-quote but then is a reference (surrounded by double curly brackets)
var nonQuotedVariableRegex = regexp.MustCompile(`:\s*[^"]\s*{{.*?}}`)

func ensureAnyTemplateStringsAreInQuotes(text string) string {
	sanitized := nonQuotedVariableRegex.ReplaceAllStringFunc(text, func(s string) string {
		s = strings.ReplaceAll(s, `{{`, `"{{`)
		s = strings.ReplaceAll(s, `}}`, `}}"`)
		return s
	})
	return sanitized
}

func putOrGet(m map[string]map[string]string, key string) map[string]string {

	if m[key] != nil {
		return m[key]
	}

	m2 := make(map[string]string)
	m[key] = m2

	return m2
}

func convert(original map[string]interface{}) (typed map[string]map[string]string, err error) {

	m2 := make(map[string]map[string]string)

	for k1, v1 := range original {
		switch v2 := v1.(type) {
		case []interface{}:
			m2Inner := putOrGet(m2, k1)
			for _, v3 := range v2 {
				switch v3 := v3.(type) {
				case map[interface{}]interface{}:
					for k3, v3 := range v3 {
						switch k3 := k3.(type) {
						case string:
							switch v3 := v3.(type) {
							case string:
								if referencesConfigJSON(k1, v3) || appearsToReferenceVariableInAnotherYaml(v3) {
									m2Inner[k3] = files.ReplacePathSeparators(v3)
								} else {
									m2Inner[k3] = v3
								}
							default:
								return m2, fmt.Errorf("cannot convert YAML on level 4: value of key '%s' has unexpected type", k3)
							}
						default:
							return m2, fmt.Errorf("cannot convert YAML on level 3: invalid key type '%s'", k3)
						}
					}
				default:
					return m2, fmt.Errorf("cannot convert YAML on level 2: %s", v3)
				}
			}
		default:
			return m2, fmt.Errorf("cannot convert YAML on level 1: value of key '%s' has unexpected type", k1)
		}
	}
	return m2, nil
}

func appearsToReferenceVariableInAnotherYaml(s string) bool {
	if containsColon(s) {
		// A path to another yaml can never ever contain a colon. Therefore, bailing out if s contains one.
		return false
	}
	if doesNotReferenceKnownVariable(s) {
		// As of right now there's only a limited number of variables that can be referenced. If s points to something else let's bail out here.
		return false
	}
	return true
}

func referencesConfigJSON(yamlSection, s string) bool {
	if yamlSection != "config" {
		return false
	}
	return strings.HasSuffix(s, ".json")
}

func containsColon(s string) bool {
	return strings.ContainsRune(s, ':')
}

var validYamlVariableReference = regexp.MustCompile(`\.(id|name)$`)

func doesNotReferenceKnownVariable(s string) bool {
	return !validYamlVariableReference.MatchString(s)
}
