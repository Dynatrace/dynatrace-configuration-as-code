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
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
)

func escapeSpecialCharactersInMap(properties map[string]interface{}, escapeFunc StringEscapeFunction) (map[string]interface{}, error) {
	escapedProperties := make(map[string]interface{}, len(properties))

	for key, value := range properties {
		escaped, err := EscapeSpecialCharactersInValue(value, escapeFunc)
		if err != nil {
			return nil, err
		}
		escapedProperties[key] = escaped
	}

	return escapedProperties, nil
}

// EscapeSpecialCharactersInValue takes a value and tries to escape any strings in it - it will walk recursively in
// case of maps/maps-of-maps of string and escape any special characters using a StringEscapeFunction.
// This is used by v1 config templating - with a simple function escaping newlines
// and v2 parameter values returns - with an escape function escaping strings to be fully JSON compliant
func EscapeSpecialCharactersInValue(value interface{}, escapeFunc StringEscapeFunction) (interface{}, error) {
	switch field := value.(type) {
	case bool:
		return field, nil
	case string:
		return escapeFunc(field)
	case map[string]string:
		return escapeCharactersForStringMap(field, escapeFunc)
	case map[string]interface{}:
		return escapeSpecialCharactersInMap(field, escapeFunc)
	default:
		log.Debug("tried to string escape value of unsupported type %v, returning unchanged", reflect.TypeOf(value))
		return value, nil
	}
}

func escapeCharactersForStringMap(properties map[string]string, escapeFunc StringEscapeFunction) (map[string]string, error) {
	escapedProperties := make(map[string]string, len(properties))

	for key, value := range properties {
		escaped, err := escapeFunc(value)
		if err != nil {
			return nil, err
		}
		escapedProperties[key] = escaped
	}

	return escapedProperties, nil
}

type StringEscapeFunction func(string) (string, error)

// FullStringEscapeFunction fully escapes any special characters in the input string, ensure it is valid for use in JSON
var FullStringEscapeFunction = escapeCharactersForJson

// escapeCharactersForJson ensures a string can be placed into a json by just marshalling it to json.
// This will escape anything that needs to be escaped - but explicitly excludes strings that are of string list format.
// Such list strings can be used to place several values into a json list and their double-quotes are needed to render
// valid json and must not be escaped. As a caveat this means any other characters aren't escaped either for lists.
// As marshalling additionally places quotes around the output these first and last characters are cut off before returning.
func escapeCharactersForJson(rawString string) (string, error) {
	b, err := marshalWithoutEscapeHTML(rawString)
	if err != nil {
		// errors should never occur for marshalling a string value - better safe than sorry if implementation details change
		return "", err
	}
	s := string(b)
	s = s[1 : len(s)-1] // marshalling places quotes around the json string which we don't want
	return s, nil
}

// marshalWithoutEscapeHTML works the same way as json.Marshal, with the exception that HTML entities (<, >, &) are
// NOT escaped.
func marshalWithoutEscapeHTML(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	buf := buffer.Bytes()
	// Encoder.Encode adds a new \n to the bytes, which json.Marshal does not
	return buf[:len(buf)-1], nil
}
