//go:build unit

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

package idutils

import (
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/v2/coordinate"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDynatraceNamespaceUUIDDoesNotPanic(t *testing.T) {
	assert.NotNil(t, dynatraceNamespaceUUID)
}

func TestGenerateUUIDFromName(t *testing.T) {
	tests := []struct {
		givenName  string
		expectUUID string
	}{
		{
			"an application detection rule",
			"51f47928-d86a-3cd0-9a2a-b0f04a1c4531",
		},
		{
			"",
			"c28406e6-ef82-362f-81d2-2da0825d64f7",
		},
		{
			"żółć",
			"fd5c1daa-6c2f-3ee1-a64d-a53df5ba7377",
		},
		{
			"abc",
			"4e198774-f86e-39ca-85ec-ac8d98a54468",
		},
		{
			"def",
			"3b55a233-aed8-3cc8-a487-7d35aaad1400",
		},
		{
			"94E6C9827A29E34D78B699D8D9D0D221",
			"41598cc6-677f-39a0-a8e8-dece5e4e27fc",
		},
		{
			"öööÄüüäÜÜÖÖÖÖ",
			"59726ed6-0bd1-35cc-8471-86d3dc44105f",
		},
	}
	for _, tt := range tests {
		t.Run("GenerateUUIDFromString("+tt.givenName+")", func(t *testing.T) {
			gotUUID := GenerateUUIDFromString(tt.givenName)

			if gotUUID != tt.expectUUID {
				t.Errorf("GenerateUUIDFromString() gotUUID = %v, want %v", gotUUID, tt.expectUUID)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	validUUID := "41598cc6-677f-39a0-a8e8-dece5e4e27fc"
	inValidUUID := "41598cc6-677f-39a0-a8e8-dece5e4e27fg"

	isUUID := IsUUID(validUUID)
	assert.Equal(t, true, isUUID)

	isUUID = IsUUID(inValidUUID)
	assert.Equal(t, false, isUUID)
}

func TestGenerateUUIDFromConfigId(t *testing.T) {
	projectUniqueId := "environment-id/project-id"
	configId := "my-config-id"
	expectedUUIDResult := "49ac3d5e-ca4a-35be-b94e-26913319bac4"

	UUIDToBeTested := GenerateUUIDFromConfigId(projectUniqueId, configId)
	assert.Equal(t, expectedUUIDResult, UUIDToBeTested)
}

func TestGenerateUUIDFromCoordinate(t *testing.T) {
	coord := coordinate.Coordinate{
		Project:  "project",
		Type:     "workflow",
		ConfigId: "id1",
	}
	expectedUUIDResult := "e8fd06bf-08ab-3a2f-9d3f-1fd66ea870a2"

	UUIDToBeTested := GenerateUUIDFromCoordinate(coord)
	assert.Equal(t, expectedUUIDResult, UUIDToBeTested)
}
