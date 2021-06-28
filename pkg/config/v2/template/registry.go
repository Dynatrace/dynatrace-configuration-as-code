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
	"encoding/json"
	"path/filepath"

	templ "text/template"

	"github.com/spf13/afero"
)

type Template interface {
	// id of the template, used as a key in the template cache
	Id() string

	// human readable name for this template, mostly used for debugging
	Name() string

	// string content of the template
	Content() string
}

type FileBasedTemplate interface {
	Template
	FilePath() string
}

// type defining a template which can be rendered
type stringTemplate struct {
	id string

	name string

	content string
}

func (t *stringTemplate) Id() string {
	return t.id
}

func (t *stringTemplate) Name() string {
	return t.name
}

func (t *stringTemplate) Content() string {
	return t.content
}

// type defining a template which can be rendered
type fileBasedTemplate struct {
	path    string
	content string
}

func (t *fileBasedTemplate) Id() string {
	return t.path
}

func (t *fileBasedTemplate) Name() string {
	return t.path
}

func (t *fileBasedTemplate) Content() string {
	return t.content
}

func (t *fileBasedTemplate) FilePath() string {
	return t.path
}

var (
	_ Template          = (*stringTemplate)(nil)
	_ FileBasedTemplate = (*fileBasedTemplate)(nil)
)

func (t *stringTemplate) MarshalJSON() ([]byte, error) {
	// Only used for debugging purpose.
	// The content of the template is dropped, since it adds too much
	// content to the debug file.
	return json.Marshal(&struct{ Name string }{Name: t.name})
}

// cache for templates so that they don't get read from disk multiple times
var templateCache = make(map[string]Template)

// cache for parsed go templates to only parse them once
var parsedTemplateCache = make(map[string]*templ.Template)

// tries to load the file at the given path and turns it into a template.
// the name of the template will be the sanitized path.
func LoadTemplate(fs afero.Fs, path string) (Template, error) {
	sanitizedPath := filepath.Clean(path)

	if template, found := templateCache[sanitizedPath]; found {
		return template, nil
	}

	data, err := afero.ReadFile(fs, sanitizedPath)

	if err != nil {
		return nil, err
	}

	content := string(data)

	template := new(fileBasedTemplate)

	*template = fileBasedTemplate{
		path:    sanitizedPath,
		content: content,
	}

	parsedTemplate, err := parseTemplate(sanitizedPath, content)

	if err != nil {
		return nil, err
	}

	templateCache[sanitizedPath] = template
	parsedTemplateCache[sanitizedPath] = parsedTemplate

	return template, nil
}

// tries to parse the given string into a template and return it
func CreateFileBasedTemplateFromString(path, content string) (Template, error) {
	sanitizedPath := filepath.Clean(path)

	if template, found := templateCache[sanitizedPath]; found {
		return template, nil
	}

	template := new(fileBasedTemplate)

	*template = fileBasedTemplate{
		path:    path,
		content: content,
	}

	parsedTemplate, err := parseTemplate(sanitizedPath, content)

	if err != nil {
		return nil, err
	}

	templateCache[sanitizedPath] = template
	parsedTemplateCache[sanitizedPath] = parsedTemplate

	return template, nil
}

// tries to parse the given string into a template and return it
func LoadTemplateFromString(id, name, content string) (Template, error) {
	if template, found := templateCache[id]; found {
		return template, nil
	}

	template := new(stringTemplate)

	*template = stringTemplate{
		id:      id,
		name:    name,
		content: content,
	}

	parsedTemplate, err := parseTemplate(id, content)

	if err != nil {
		return nil, err
	}

	templateCache[id] = template
	parsedTemplateCache[id] = parsedTemplate

	return template, nil
}

func parseTemplate(id, content string) (*templ.Template, error) {
	return templ.New(id).Option("missingkey=error").Parse(content)
}
