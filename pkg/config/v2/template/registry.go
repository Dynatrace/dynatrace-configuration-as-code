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
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/util/log"
	"path/filepath"

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
	_ FileBasedTemplate = (*fileBasedTemplate)(nil)
)

// tries to load the file at the given path and turns it into a template.
// the name of the template will be the sanitized path.
func LoadTemplate(fs afero.Fs, path string) (Template, error) {
	sanitizedPath := filepath.Clean(path)

	log.Debug("Loading template for %s", sanitizedPath)

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

	return template, nil
}

// tries to parse the given string into a template and return it
func CreateTemplateFromString(path, content string) (Template, error) {
	sanitizedPath := filepath.Clean(path)

	log.Debug("Loading file-based template for %s", sanitizedPath)

	template := new(fileBasedTemplate)

	*template = fileBasedTemplate{
		path:    path,
		content: content,
	}

	return template, nil
}
