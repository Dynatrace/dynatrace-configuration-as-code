/*
 * @license
 * Copyright 2025 Dynatrace LLC
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
)

// UseGoTemplatesForDoubleCurlyBraces replaces each occurrence of "{{" with "{{`{{`}}" and each occurrence of "}}" with "{{`}}`}}".
// This ensures that when the returned string is used to render templates, e.g. during deployment, the "{{" and "}}" are not misinterpreted.
func UseGoTemplatesForDoubleCurlyBraces(src []byte) []byte {
	src = bytes.ReplaceAll(src, []byte("{{"), []byte("{{`{{`")) // replace is divided in 2 steps to avoid replacing of closing brackets in the next step
	src = bytes.ReplaceAll(src, []byte("}}"), []byte("{{`}}`}}"))
	src = bytes.ReplaceAll(src, []byte("{{`{{`"), []byte("{{`{{`}}"))
	return src
}
