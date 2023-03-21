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

package v2

import (
	"regexp"
)

// matches any non-alphanumerical chars including -, _, .
var namePattern = regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)

const MaxFilenameLengthWithoutFileExtension = 254

// Sanitize removes special characters, limits to max 254 characters in name, no special characters except '-', '_', and '.'
func Sanitize(name string) string {
	processedString := namePattern.ReplaceAllString(name, "")

	runes := []rune(processedString)
	if len(runes) > MaxFilenameLengthWithoutFileExtension {
		processedString = string(runes[:MaxFilenameLengthWithoutFileExtension])
	}

	return processedString
}
