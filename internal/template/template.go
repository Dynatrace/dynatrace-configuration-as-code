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
	"bytes"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/errutils"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"os"
	"strings"
	"text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template

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
// In v2 config templating this is replaced by renderer.go/Render
func (t *templateImpl) ExecuteTemplate(data map[string]string) (string, error) {

	tpl := bytes.Buffer{}

	dataForTemplating := addEnvVars(data)

	dataForTemplating, err := EscapeSpecialCharacters(dataForTemplating)
	if errutils.CheckError(err, "Failed to prepare config properties for templating") {
		return "", err
	}

	err = t.template.Execute(&tpl, dataForTemplating)
	if errutils.CheckError(err, "Could not execute template") {
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
			log.Info("Environment variable %s also defined as property. Was that your intention?", split[0])
		}

		envVars[split[0]] = split[1]
	}

	return data
}
