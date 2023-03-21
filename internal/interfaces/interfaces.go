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

package interfaces

import (
	"reflect"

	"github.com/dynatrace/dynatrace-configuration-as-code/internal/string_util"
)

func GetDynamicFieldFromObject(object interface{}, field string) reflect.Value {
	reflection := reflect.ValueOf(object)

	fieldValue := reflect.Indirect(reflection).FieldByName(field)

	// We are providing uncapitalized fields from json maps
	// But GoLang forces capitalized for unmarshalling
	// Let's try a capitalized first letter
	if IsInvalidReflectionValue(fieldValue) {
		field = string_util.CapitalizeFirstLetter(field)
		fieldValue = reflect.Indirect(reflection).FieldByName(field)
	}
	return fieldValue
}

func GetDynamicFieldFromMapReflection(reflection reflect.Value, field string) reflect.Value {
	return reflection.MapIndex(reflect.ValueOf(field))
}

func HasSpecificFieldValueInSlice(slice reflect.Value, field string, searchFieldValue string) bool {

	for i := 0; i < slice.Len(); i++ {
		element := slice.Index(i)
		if IsInvalidReflectionValue(element) {
			continue
		}

		idValue := GetDynamicFieldFromMapReflection(element, field)
		if IsInvalidReflectionValue(idValue) {
			continue
		}
		if idValue.Interface().(string) == searchFieldValue {
			return true
		}
	}

	return false

}

func IsInvalidReflectionValue(value reflect.Value) bool {
	if value.Kind() == reflect.Invalid {
		return true
	} else {
		return false
	}
}
