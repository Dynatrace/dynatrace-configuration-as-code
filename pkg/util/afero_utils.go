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

package util

import (
	"regexp"

	"github.com/spf13/afero"
)

// CreateTestFileSystem creates a virtual filesystem with 2 layers.
// The first layer allows to read file from the disk
// the second layer allows to modify files on a virtual filesystem
func CreateTestFileSystem() afero.Fs {
	base := afero.NewOsFs()
	baseLayer := afero.NewReadOnlyFs(base)
	return afero.NewCopyOnWriteFs(baseLayer, afero.NewMemMapFs())
}

// matches any non-alphanumerical chars including -, _, .
var namePattern = regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)

// matches any non-alphanumerical chars including _
var templatePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

const MaxFilenameLengthWithoutFileExtension = 254

// SanitizeName removes special characters, limits to max 254 characters in name, no special characters except '-', '_', and '.'
func SanitizeName(name string) string {
	processedString := namePattern.ReplaceAllString(name, "")

	runes := []rune(processedString)
	if len(runes) > MaxFilenameLengthWithoutFileExtension {
		processedString = string(runes[:MaxFilenameLengthWithoutFileExtension])
	}

	return processedString
}

// SanitizeTemplateVar removes all except alphanumerical chars and underscores (_)
func SanitizeTemplateVar(templateVarName string) string {
	return templatePattern.ReplaceAllString(templateVarName, "")
}
