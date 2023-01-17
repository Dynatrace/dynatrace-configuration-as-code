// @license
// Copyright 2022 Dynatrace LLC
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

package client

import (
	reflect "reflect"
	"unicode"
)

const pathEntitiesObjects = "/api/v2/entities"
const pathEntitiesTypes = "/api/v2/entityTypes"

// Full export would be: const defaultListEntitiesFields = "+lastSeenTms,+firstSeenTms,+tags,+managementZones,+toRelationships,+fromRelationships,+icon,+properties"
// Using smaller export for faster processing, using less memory
const defaultListEntitiesFields = "+lastSeenTms,+firstSeenTms"

var extraEntitiesFields = map[string][]string{
	"properties":      {"detectedName", "oneAgentCustomHostName", "ipAddress"},
	"toRelationships": {"isSiteOf", "isClusterOfHost"},
}

func getEntitiesTypeFields(entitiesType EntitiesType, ignoreProperties []string) string {
	typeFields := defaultListEntitiesFields

	for topField, subFieldList := range extraEntitiesFields {

		if contains(ignoreProperties, topField) {
			continue
		}
		fieldSliceObject := getDynamicFieldFromObject(entitiesType, topField)

		if isInvalidReflectionValue(fieldSliceObject) {

		} else {
			for _, subField := range subFieldList {
				if contains(ignoreProperties, subField) {
					continue
				}
				if hasSpecificFieldValueInSlice(fieldSliceObject, "id", subField) {
					typeFields = typeFields + ",+" + topField + "." + subField
				}
			}
		}
	}

	return typeFields
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getDynamicFieldFromObject(object interface{}, field string) reflect.Value {
	reflection := reflect.ValueOf(object)

	fieldValue := reflect.Indirect(reflection).FieldByName(field)

	// We are providing uncapitalized fields from json maps
	// But GoLang forces capitalized for unmarshalling
	// Let's try a capitalized first letter
	if isInvalidReflectionValue(fieldValue) {
		field = capitalizeFirstLetter(field)
		fieldValue = reflect.Indirect(reflection).FieldByName(field)
	}
	return fieldValue
}

func getDynamicFieldFromMapReflection(reflection reflect.Value, field string) reflect.Value {
	return reflection.MapIndex(reflect.ValueOf(field))
}

func hasSpecificFieldValueInSlice(slice reflect.Value, field string, searchFieldValue string) bool {

	for i := 0; i < slice.Len(); i++ {
		element := slice.Index(i)
		if isInvalidReflectionValue(element) {
			continue
		}

		idValue := getDynamicFieldFromMapReflection(element, field)
		if isInvalidReflectionValue(idValue) {
			continue
		}
		if idValue.Interface().(string) == searchFieldValue {
			return true
		}
	}

	return false

}

func capitalizeFirstLetter(str string) string {
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	capitalizedString := string(runes)

	return capitalizedString
}

func isInvalidReflectionValue(value reflect.Value) bool {
	if value.Kind() == reflect.Invalid {
		return true
	} else {
		return false
	}
}
