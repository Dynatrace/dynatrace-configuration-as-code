/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package secret

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

var jsonKeysRegex = regexp.MustCompile(`"(\w+)"\s*:`)

const EnvMonacoSupportArchiveMaskKeys = "MONACO_SUPPORT_ARCHIVE_MASKED_KEYS"

var defaultMaskedKeys = []string{
	"access",
	"credential",
	"key",
	"password",
	"secret",
	"token",
}

func maskedKeysFromEnv() []string {
	k, found := os.LookupEnv(EnvMonacoSupportArchiveMaskKeys)
	if !found {
		return defaultMaskedKeys
	}
	// return array of keys. If k is empty ("") then an empty slice is returned
	return strings.FieldsFunc(k, func(c rune) bool { return c == ',' })

}

func Mask(data []byte) []byte {
	keysToMask := maskedKeysFromEnv()
	if len(keysToMask) == 0 {
		return data
	}

	jsonKeys := getJsonKeys(string(data))
	keysToSearch := make([]string, 0)
	for _, k := range jsonKeys {
		for _, kk := range keysToMask {
			if strings.Contains(strings.ToLower(k), strings.ToLower(kk)) {
				keysToSearch = append(keysToSearch, kk)
			}
		}
	}

	if len(keysToSearch) == 0 {
		return data
	}

	return mask(data, keysToSearch)
}
func mask(jsonStr []byte, keysToSearch []string) []byte {
	var data interface{}
	err := json.Unmarshal(jsonStr, &data)
	if err != nil {
		return []byte(`"NON-JSON CONTENT"`)
	}

	maskRecursive(data, keysToSearch)

	maskedJson, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}

	return maskedJson
}

func getJsonKeys(json string) []string {
	matches := jsonKeysRegex.FindAllStringSubmatch(json, -1)
	jsonKeys := make([]string, len(matches))
	for i, match := range matches {
		if len(match) > 1 {
			jsonKeys[i] = strings.Trim(match[1], `"`)
		}
	}
	return jsonKeys
}

func maskRecursive(data interface{}, keys []string) {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			if _, ok := val.(string); ok {
				for _, key := range keys {
					if strings.Contains(strings.ToLower(k), strings.ToLower(key)) {
						v[k] = "########"
					}
				}
			}
			maskRecursive(val, keys)
		}
	case []interface{}:
		for i := range v {
			maskRecursive(v[i], keys)
		}
	}
}
