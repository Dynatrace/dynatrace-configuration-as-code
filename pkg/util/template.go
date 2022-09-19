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
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/afero"
)

// Template wraps the underlying templating logic and provides a means of setting config values just on one place.
// It is intended to be language-agnostic, the file type does not matter (yaml, json, ...)
type Template interface {
	ExecuteTemplate(data map[string]string) (string, error)
}

type templateImpl struct {
	template *template.Template
}

// NewTemplateFromString creates a new template for the given string content
func NewTemplateFromString(name string, content string) (Template, error) {

	templ := template.New(name).Option("missingkey=error")
	templ, err := templ.Parse(content)

	if err != nil {
		return nil, err
	}

	return newTemplate(templ), nil
}

// NewTemplate creates a new template for the given file
func NewTemplate(fs afero.Fs, fileName string) (Template, error) {
	data, err := afero.ReadFile(fs, fileName)

	if err != nil {
		return nil, err
	}

	return NewTemplateFromString(fileName, string(data))
}

func newTemplate(templ *template.Template) Template {

	// Fail fast on missing variable (key):
	templ = templ.Option("missingkey=error")

	return &templateImpl{
		template: templ,
	}
}

// ExecuteTemplate executes the given template. It fills the placeholder variables in the template with the strings
// in the data map. Additionally, it resolves all environment variables present in the template.
// Important: if a variable present in the template has no corresponding entry in the data map, this method will throw
// an error
func (t *templateImpl) ExecuteTemplate(data map[string]string) (string, error) {

	tpl := bytes.Buffer{}

	dataForTemplating := addEnvVars(data)

	dataForTemplating, err := escapeSpecialCharacters(dataForTemplating)
	if CheckError(err, "Failed to prepare config properties for templating") {
		return "", err
	}

	err = t.template.Execute(&tpl, dataForTemplating)
	if CheckError(err, "Could not execute template") {
		return "", err
	}

	return tpl.String(), nil
}

func addEnvVars(properties map[string]string) map[string]interface{} {

	data := make(map[string]interface{})

	for k := range properties {
		data[k] = properties[k]
	}

	envVars := make(map[string]string)
	data["Env"] = envVars

	for _, v := range os.Environ() {
		split := strings.SplitN(v, "=", 2)
		if len(split) != 2 {
			continue
		}

		if _, ok := properties[split[0]]; ok {
			Log.Info("Environment variable %s also defined as property. Was that your intention?", split[0])
		}

		envVars[split[0]] = split[1]
	}

	return data
}

// escapeSpecialCharacters walks recursively though the map and escapes all special characters that can't just be written to the
// json template. characters that will be escaped: newlines (\n), double quotes (\")
func escapeSpecialCharacters(properties map[string]interface{}) (map[string]interface{}, error) {

	escapedProperties := make(map[string]interface{}, len(properties))

	for key, value := range properties {

		switch field := value.(type) {
		case string:
			escaped, err := escapeCharactersForJson(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		case map[string]string:
			escaped, err := escapeNewlineCharactersForStringMap(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		case map[string]interface{}:
			escaped, err := escapeSpecialCharacters(field)
			if err != nil {
				return nil, err
			}
			escapedProperties[key] = escaped
		default:
			Log.Debug("Unknown value type %v in property %v.", reflect.TypeOf(value), key)
		}
	}

	return escapedProperties, nil
}

func escapeNewlineCharactersForStringMap(properties map[string]string) (map[string]string, error) {
	escapedProperties := make(map[string]string, len(properties))

	for key, value := range properties {
		escaped, err := escapeCharactersForJson(value)
		if err != nil {
			return nil, err
		}
		escapedProperties[key] = escaped
	}

	return escapedProperties, nil
}

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

// pattern matching strings of the format '"value", "value", ...' which are sometimes used to set lists into JSON templates
// these must generally not have their quotes escaped as their JSON template is usually not valid with these values
var listPattern = regexp.MustCompile(`(?:\s*".*?"\s*,\s*".*?"\s*,?)+`)

func isListDefinition(s string) bool {
	return listPattern.MatchString(s)
}
