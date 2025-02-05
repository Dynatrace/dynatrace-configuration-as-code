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

var (
	_ Template = (*InMemoryTemplate)(nil)
)

// InMemoryTemplate is a JSON Template created in-memory. This is generally used by downloads and conversions, where
// a Template is created on the fly. Deployments generally use FileBasedTemplate instead.
// An InMemoryTemplate may define a dedicated path to write it to when persisting - converted Template are create with a
// path, download ones are not.
type InMemoryTemplate struct {
	id      string
	content string
	// optional path we'd like this Template to be written to if it's persisted
	path *string
}

func (t *InMemoryTemplate) ID() string {
	return t.id
}

func (t *InMemoryTemplate) Content() (string, error) {
	return t.content, nil
}

func (t *InMemoryTemplate) UpdateContent(newContent string) error {
	t.content = newContent
	return nil
}

// FilePath returns the optional path this Template should be written to if it's persisted. If an InMemoryTemplate has
// not defined a path nil will be returned and the file may be persisted anywhere. Generally a converted v1 Template will
// have a defined FilePath, while a downloaded template does not.
func (t *InMemoryTemplate) FilePath() *string {
	return t.path
}

// NewInMemoryTemplate creates a new InMemoryTemplate without a dedicated path it should be written to if persisted.
// To create an InMemoryTemplate with a fixed target path, use NewInMemoryTemplateWithPath.
func NewInMemoryTemplate(id, content string) Template {
	return &InMemoryTemplate{
		id:      id,
		content: content,
	}
}

// NewInMemoryTemplateWithPath creates a new InMemoryTemplate with a dedicated path it should be written to if persisted.
// To create a simple InMemoryTemplate without filepath, use NewInMemoryTemplate.
// Deprecated: Don't use anymore: Only used written to in tests, and used once for reading.
func NewInMemoryTemplateWithPath(filepath, content string) Template {
	return &InMemoryTemplate{
		path:    &filepath,
		content: content,
	}
}
