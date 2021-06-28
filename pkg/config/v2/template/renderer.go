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
)

// tries to render a given template with the given properties and returns the
// resulting string. if any error occurs during rendering, an error is returned.
func Render(template Template, properties map[string]interface{}) (string, error) {
	parsedTemplate, found := parsedTemplateCache[template.Id()]

	// if we do not find the template in the template cache, it means that it was
	// somehow instantiated without calling the template registry functions.
	if !found {
		return "", fmt.Errorf("trying to render unknown template `%s`. this should not happen and is likely a bug",
			template.Name())
	}

	result := bytes.Buffer{}

	err := parsedTemplate.Execute(&result, properties)

	if err != nil {
		return "", err
	}

	return result.String(), nil
}
