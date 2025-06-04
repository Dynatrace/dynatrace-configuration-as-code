//go:build integration || cleanup || download_restore || unit || nightly

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

package runner

import (
	"fmt"
	"strings"
)

func AddSuffix(name string, suffix string) string {
	return name + "_" + suffix
}

func GetAddSuffixFunction(suffix string) func(line string) string {
	var f = func(name string) string {
		return AddSuffix(name, suffix)
	}
	return f
}

func ReplaceName(line string, idChange func(string) string) string {
	if strings.HasSuffix(line, "#monaco-test:no-replace") {
		return line
	}

	if strings.Contains(line, "env-token-name:") {
		return line
	}

	if !strings.Contains(line, "name:") {
		return line
	}

	trimmed := strings.TrimSpace(line)
	split := strings.SplitN(trimmed, ":", 2)

	key := split[0]
	val := split[1]

	if !isNameKey(key) {
		return line
	}

	name := strings.TrimSpace(val)

	if name == "" { //line only contained the name, can't do anything here and probably a non-shorthand v2 reference
		return line
	}

	if strings.HasPrefix(name, "\"") || strings.HasPrefix(name, "'") {
		name = name[1 : len(name)-1]
	}

	// Dependencies are not substituted
	if isV2Dependency(name) {
		return line
	}

	replaced := strings.ReplaceAll(line, name, idChange(name))
	return replaced

}

func isNameKey(key string) bool {
	key = strings.TrimSpace(key)
	key = strings.TrimPrefix(key, "-")
	key = strings.TrimSpace(key)
	return key == "name"
}

func ReplaceId(line string, idChange func(string) string) string {
	if strings.HasSuffix(line, "#monaco-test:no-replace") {
		return line
	}

	if strings.Contains(line, "id:") || strings.Contains(line, "configId:") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") {
			trimmed = trimmed[1:]
			trimmed = strings.TrimSpace(trimmed)
		}
		var id string
		if strings.HasPrefix(trimmed, "id:") {
			withoutPrefix := strings.TrimLeft(trimmed, "id:")
			id = strings.TrimSpace(withoutPrefix)
		} else if strings.HasPrefix(trimmed, "configId:") {
			withoutPrefix := strings.TrimLeft(trimmed, "configId:")
			id = strings.TrimSpace(withoutPrefix)
		}
		if id == "" { //line only contained the name, can't do anything here and probably a non-shorthand v2 reference
			return line
		}
		id = strings.Trim(id, `"'`)
		replaced := strings.ReplaceAll(line, id, idChange(id))
		return replaced
	}

	entries := strings.SplitN(line, ":", 2)
	if len(entries) != 2 { //not a key:value pair
		return line
	}
	key := entries[0]
	property := entries[1]

	if strings.TrimSpace(key) == "values" { //very likely list-type array, don't touch
		return line
	}

	if isV2Dependency(property) {
		property := strings.TrimSpace(property)
		property = strings.Trim(property, "[]")

		ref := strings.Split(property, ",")
		config := ref[len(ref)-2] // 2nd to last is cfgID
		config = strings.TrimSpace(config)
		config = strings.Trim(config, `"'`)

		ref[len(ref)-2] = fmt.Sprintf(`"%s"`, idChange(config))
		return fmt.Sprintf("%s: [%s]", key, strings.Join(ref, ","))
	}
	return line
}

func isV2Dependency(name string) bool {
	s := strings.TrimSpace(name)
	if !(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		return false
	}
	s = strings.Trim(s, "[]")
	if s == "" {
		return false
	}
	split := strings.Split(s, ",")
	if len(split) < 2 || len(split) > 4 {
		// does not contain cfgID or is too long for ref
		return false
	}
	return true
}
