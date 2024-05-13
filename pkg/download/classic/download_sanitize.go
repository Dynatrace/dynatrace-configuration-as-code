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

package classic

import "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/api"

var apiSanitizeFunctions = map[string]func(properties map[string]interface{}) map[string]interface{}{
	api.ServiceDetectionFullWebService:   removeOrderProperty,
	api.ServiceDetectionFullWebRequest:   removeOrderProperty,
	api.ServiceDetectionOpaqueWebService: removeOrderProperty,
	api.ServiceDetectionOpaqueWebRequest: removeOrderProperty,
	api.MaintenanceWindow: func(properties map[string]interface{}) map[string]interface{} {
		if s, ok := properties["scope"].(map[string]interface{}); ok {
			var emptyEntities, emptyMatches bool
			if entities, ok := s["entities"].([]interface{}); ok && len(entities) == 0 {
				properties = removeByPath(properties, []string{"scope", "entities"})
				emptyEntities = true
			}
			if matches, ok := s["matches"].([]interface{}); ok && len(matches) == 0 {
				properties = removeByPath(properties, []string{"scope", "matches"})
				emptyMatches = true
			}
			if emptyEntities && emptyMatches {
				properties = removeByPath(properties, []string{"scope"})
			}
		}

		return properties
	},
	api.UserActionAndSessionPropertiesMobile: func(properties map[string]interface{}) map[string]interface{} {
		return removeByPath(properties, []string{"key"})
	},
}

func sanitizeProperties(properties map[string]interface{}, apiId string) map[string]interface{} {
	properties = removeIdentifyingProperties(properties, apiId)
	properties = removePropertiesNotAllowedOnUpload(properties, apiId)

	// user action and session properties configs have the "key" as name, hence
	// we must avoid overwriting the name property with wrong values
	if apiId == api.UserActionAndSessionPropertiesMobile {
		return properties
	}

	return replaceTemplateProperties(properties)
}

func removeIdentifyingProperties(dat map[string]interface{}, apiId string) map[string]interface{} {
	dat = removeByPath(dat, []string{"metadata"})
	dat = removeByPath(dat, []string{"id"})
	dat = removeByPath(dat, []string{"applicationId"})
	dat = removeByPath(dat, []string{"identifier"})
	dat = removeByPath(dat, []string{"rules", "id"})
	dat = removeByPath(dat, []string{"rules", "methodRules", "id"})
	dat = removeByPath(dat, []string{"conversionGoals", "id"})

	// After manual inspection, it appears that only 'calculated-metrics-service' needs to still keep the entityId.
	// The other APIs are self-referencing (e.g. HTTP-CHECK-0123 has entityId set to its own ID).
	if apiId != api.CalculatedMetricsService {
		dat = removeByPath(dat, []string{"entityId"})
	}

	return dat
}

func removePropertiesNotAllowedOnUpload(properties map[string]interface{}, apiId string) map[string]interface{} {
	if specificSanitizer := apiSanitizeFunctions[apiId]; specificSanitizer != nil {
		return specificSanitizer(properties)
	}
	return properties
}

func removeOrderProperty(properties map[string]interface{}) map[string]interface{} {
	return removeByPath(properties, []string{"order"})
}

func removeByPath(dat map[string]interface{}, key []string) map[string]interface{} {
	if len(key) == 0 || dat == nil || dat[key[0]] == nil {
		return dat
	}

	if len(key) == 1 {
		delete(dat, key[0])
		return dat
	}

	if field, ok := dat[key[0]].(map[string]interface{}); ok {
		dat[key[0]] = removeByPath(field, key[1:])
		return dat
	}

	if arrayOfFields, ok := dat[key[0]].([]interface{}); ok {
		for i := range arrayOfFields {
			if field, ok := arrayOfFields[i].(map[string]interface{}); ok {
				arrayOfFields[i] = removeByPath(field, key[1:])
			}
		}

		dat[key[0]] = arrayOfFields
	}
	return dat
}

func replaceTemplateProperties(dat map[string]interface{}) map[string]interface{} {
	const nameTemplate = "{{.name}}"

	if dat["name"] != nil {
		dat["name"] = nameTemplate
	} else if dat["displayName"] != nil {
		dat["displayName"] = nameTemplate
	}

	// replace dashboard name
	if dat["dashboardMetadata"] != nil {
		if t, ok := dat["dashboardMetadata"].(map[string]interface{}); ok && t["name"] != "" {
			t["name"] = nameTemplate
			dat["dashboardMetadata"] = t
		}
	}

	return dat
}
