/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package util

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// UnmarshalYaml takes the contents of a yaml file and converts it to a map[string]map[string]string
// The yaml file should have the following format:
//
// some-name-1:
//  - list-key-1: "list-entry-1"
//  - list-key-2: "list-entry-2"
// some-name-2:
//  - list-key-1: "list-entry-1"
//
func UnmarshalYaml(text string, fileName string) (error, map[string]map[string]string) {

	template, err := NewTemplateFromString(fileName, text)
	if err != nil {
		return err, make(map[string]map[string]string)
	}

	text, err = template.ExecuteTemplate(make(map[string]string))
	if err != nil {
		return err, make(map[string]map[string]string)
	}

	m := make(map[string]interface{})

	err = yaml.Unmarshal([]byte(text), &m)
	FailOnError(err, "Failed to unmarshal yaml\n"+text+"\nerror:")

	err, typed := convert(m)
	FailOnError(err, "YAML file "+fileName+" could not be parsed")

	return nil, typed
}

func ReplacePathSeparators(path string) (newPath string) {
	newPath = strings.ReplaceAll(path, "\\", string(os.PathSeparator))
	newPath = strings.ReplaceAll(newPath, "/", string(os.PathSeparator))
	return newPath
}

func putOrGet(m map[string]map[string]string, key string) map[string]string {

	if m[key] != nil {
		return m[key]
	}

	m2 := make(map[string]string)
	m[key] = m2

	return m2
}

func convert(original map[string]interface{}) (err error, typed map[string]map[string]string) {

	m2 := make(map[string]map[string]string)
	err = errors.New("cannot convert YAML")

	for k1, v1 := range original {
		switch v2 := v1.(type) {
		case []interface{}:
			m2Inner := putOrGet(m2, k1)
			for _, v3 := range v2 {
				switch v3 := v3.(type) {
				case map[interface{}]interface{}:
					for k3, v3 := range v3 {
						switch k3 := k3.(type) {
						case string:
							switch v3 := v3.(type) {
							case string:
								if startsWithPathSeparator(v3) {
									m2Inner[k3] = ReplacePathSeparators(v3)
								} else {
									m2Inner[k3] = v3
								}
							default:
								return err, m2
							}
						default:
							return err, m2
						}
					}
				default:
					return err, m2
				}
			}
		default:
			return err, m2
		}
	}
	return nil, m2
}

func startsWithPathSeparator(s string) bool {
	firstChar := string(s[0])
	return firstChar == "/" || firstChar == "\\"
}
