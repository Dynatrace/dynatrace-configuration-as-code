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
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/internal/log"
	"path/filepath"

	"github.com/spf13/afero"
)

// Template is the main interface of a configuration payload that may contain template references (using Go Templates)
// The only implementation used in monaco is the FilebasedTemplate, but this interface is meant to enable any usecase
// where the content of a configuration is not coming from a file - e.g. a possible use as a library in a terraform
// provider, which could implement its own extension of a Template that turns an object in TF configuration language into
// deployable JSON payload when Content() is called.
type Template interface {
	// id of the template, used as a key in the template cache
	Id() string

	// human readable name for this template, mostly used for debugging
	Name() string

	// string content of the template
	Content() string

	// UpdateContent sets the content of the template to the new provided one
	UpdateContent(newContent string)
}

type DownloadTemplate struct {
	id, name, content string
}

// FileBasedTemplate is the usual (only) type of config template monaco uses
// This is the usual API payload JSON file
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

func (t *fileBasedTemplate) UpdateContent(newContent string) {
	t.content = newContent
}

func (d *DownloadTemplate) Id() string {
	return d.id
}

func (d *DownloadTemplate) Name() string {
	return d.name
}

func (d *DownloadTemplate) Content() string {
	return d.content
}

func (d *DownloadTemplate) UpdateContent(newContent string) {
	d.content = newContent
}

// Force the compiler to check whether the structs implement the interfaces
var (
	_ FileBasedTemplate = (*fileBasedTemplate)(nil)
	_ Template          = (*fileBasedTemplate)(nil)
	_ Template          = (*DownloadTemplate)(nil)
)

// tries to load the file at the given path and turns it into a template.
// the name of the template will be the sanitized path.
func LoadTemplate(fs afero.Fs, path string) (Template, error) {
	sanitizedPath := filepath.Clean(path)

	log.Debug("Loading template for %s", sanitizedPath)

	data, err := afero.ReadFile(fs, sanitizedPath)

	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	content := string(data)

	template := fileBasedTemplate{
		path:    sanitizedPath,
		content: content,
	}

	return &template, nil
}

// tries to parse the given string into a template and return it
func CreateTemplateFromString(path, content string) Template {
	sanitizedPath := filepath.Clean(path)

	log.Debug("Loading file-based template for %s", sanitizedPath)

	template := fileBasedTemplate{
		path:    path,
		content: content,
	}

	return &template
}

func NewDownloadTemplate(id, name, content string) Template {
	return &DownloadTemplate{
		name:    name,
		content: content,
		id:      id,
	}
}
