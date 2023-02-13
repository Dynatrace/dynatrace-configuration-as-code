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
	"github.com/dynatrace/dynatrace-configuration-as-code/pkg/util/interfaces"
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
		fieldSliceObject := interfaces.GetDynamicFieldFromObject(entitiesType, topField)

		if interfaces.IsInvalidReflectionValue(fieldSliceObject) {
		} else {
			for _, subField := range subFieldList {
				if contains(ignoreProperties, subField) {
					continue
				}
				if interfaces.HasSpecificFieldValueInSlice(fieldSliceObject, "id", subField) {
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
