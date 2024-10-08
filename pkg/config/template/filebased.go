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
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/cache"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"github.com/spf13/afero"
	"path/filepath"
	"strings"
)

var (
	_ Template = (*FileBasedTemplate)(nil)
)

// FileBasedTemplate is a JSON Template stored in a file - when it's Template.Content is accessed, that file is read.
// This is the usual type of Template monaco uses.
type FileBasedTemplate struct {
	// fs is the file system the to read the template file from
	fs afero.Fs
	// path of the template file
	path string
}

func (t *FileBasedTemplate) ID() string {
	return t.path
}

func (t *FileBasedTemplate) Content() (string, error) {
	b, err := afero.ReadFile(t.fs, t.path)
	if err != nil {
		return "", fmt.Errorf("failed to read template content: %w", err)
	}
	return string(b), nil
}

func (t *FileBasedTemplate) FilePath() string {
	return t.path
}

func (t *FileBasedTemplate) UpdateContent(newContent string) error {
	f, err := t.fs.Open(t.path)
	if err != nil {
		return fmt.Errorf("failed to update template content: %w", err)
	}

	if _, err = f.WriteString(newContent); err != nil {
		return fmt.Errorf("failed to update template content: %w", err)
	}
	return nil
}

// NewFileTemplate creates a FileBasedTemplate for a given afero.Fs and filepath.
// If the file can not be accessed an error will be returned.
func NewFileTemplate(fs afero.Fs, tplCache cache.Cache[FileBasedTemplate], path string) (Template, error) {
	sanitizedPath := filepath.Clean(strings.ReplaceAll(path, `\`, `/`))

	if tmpl, ok := tplCache.Get(sanitizedPath); ok {
		return &tmpl, nil
	}

	log.Debug("Loading template for %s", sanitizedPath)

	if exists, err := afero.Exists(fs, sanitizedPath); err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	} else if !exists {
		return nil, fmt.Errorf(`template file "%s" does not exist`, sanitizedPath)
	}

	template := FileBasedTemplate{
		fs:   fs,
		path: sanitizedPath,
	}

	tplCache.Set(sanitizedPath, template)
	return &template, nil
}
