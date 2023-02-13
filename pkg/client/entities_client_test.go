//go:build unit

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

package client

import (
	"encoding/json"
	"gotest.tools/assert"
	"testing"
)

func Test_fields(t *testing.T) {
	APMSecurityGatewayJSON := `{
		"type": "APM_SECURITY_GATEWAY", 
		"displayName": "ActiveGate", 
		"dimensionKey": "dt.entity.apm_security_gateway", 
		"entityLimitExceeded": false, 
		"properties": [
			{"id": "awsNameTag", "type": "String", "displayName": "awsNameTag"}, 
			{"id": "boshName", "type": "String", "displayName": "boshName"}, 
			{"id": "conditionalName", "type": "String", "displayName": "conditionalName"}, 
			{"id": "customizedName", "type": "String", "displayName": "customizedName"}, 
			{"id": "detectedName", "type": "String", "displayName": "detectedName"}, 
			{"id": "gcpZone", "type": "String", "displayName": "gcpZone"}, 
			{"id": "isContainerDeployment", "type": "Boolean", "displayName": "isContainerDeployment"}, 
			{"id": "oneAgentCustomHostName", "type": "String", "displayName": "oneAgentCustomHostName"}
		], 
		"tags": "List", 
		"managementZones": "List", 
		"fromRelationships": [
			{"id": "isLocatedIn", "toTypes": ["SYNTHETIC_LOCATION"]}
		], 
		"toRelationships": []
	}`

	tests := []struct {
		name             string
		EntitiesTypeJSON string
		ignoreProperties []string
		want             string
	}{
		{
			"Extract Values - without ignore",
			APMSecurityGatewayJSON,
			[]string{},
			"+lastSeenTms,+firstSeenTms,+properties.detectedName,+properties.oneAgentCustomHostName",
		},
		{
			"Extract Values - ignore 1 property",
			APMSecurityGatewayJSON,
			[]string{"detectedName"},
			"+lastSeenTms,+firstSeenTms,+properties.oneAgentCustomHostName",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			entitiesType := EntitiesType{}
			json.Unmarshal([]byte(tt.EntitiesTypeJSON), &entitiesType)
			fields := getEntitiesTypeFields(entitiesType, tt.ignoreProperties)
			assert.Equal(t, fields, tt.want)
		})
	}
}
