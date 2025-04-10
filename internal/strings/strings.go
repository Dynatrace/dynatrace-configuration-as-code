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

package strings

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

func ToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

// CapitalizeFirstRuneInString returns the specified string with the first rune in uppercase. If the first rune cannot be extracted, the string is returned unchanged.
func CapitalizeFirstRuneInString(s string) string {
	firstRune, width := utf8.DecodeRuneInString(s)
	if firstRune == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(firstRune)) + s[width:]
}
