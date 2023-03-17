// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package template

import (
	"bytes"
	"fmt"
	templ "text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template
)

// Render tries to render a given template with the given properties and returns the
// resulting string. if any error occurs during rendering, an error is returned.
func Render(template Template, properties map[string]interface{}) (string, error) {
	parsedTemplate, err := ParseTemplate(template.Id(), template.Content())

	if err != nil {
		return "", fmt.Errorf("failure trying to render template %s: %w", template.Name(), err)
	}

	result := bytes.Buffer{}

	err = parsedTemplate.Execute(&result, properties)
	if err != nil {
		return "", fmt.Errorf("failure trying to render template %s: %w", template.Name(), err)
	}

	return result.String(), nil
}

// ParseTemplate creates go Template with the given id from the given string content
// in any error occurs creating the template, an erro is returned
func ParseTemplate(id, content string) (*templ.Template, error) {
	return templ.New(id).Option("missingkey=error").Parse(content)
}
