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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMasker_Mask(t *testing.T) {

	t.Setenv(EnvMonacoSupportArchiveMaskKeys, "searchKey1,searchKey2,searchKey3")

	tests := []struct {
		name    string
		jsonStr string
		want    string
	}{
		{
			name:    "Masking field value",
			jsonStr: `{"searchKey1":"1234","username":"user1"}`,
			want:    `{"searchKey1":"########","username":"user1"}`,
		},
		{
			name:    "Masking nested field value",
			jsonStr: `{"user":{"searchKey1":"1234","username":"user1"}}`,
			want:    `{"user":{"searchKey1":"########","username":"user1"}}`,
		},
		{
			name:    "Not masking nested complex field value",
			jsonStr: `{"user":{"searchKey1":{"user":{"searchKey1":"1234","username":"user1"}},"username":"user1"}}`,
			want:    `{"user":{"searchKey1":{"user":{"searchKey1":"########","username":"user1"}},"username":"user1"}}`,
		},
		{
			name:    "Masking multiple values",
			jsonStr: `{"searchKey1":"1234","user":{"searchKey2":{"user":{"searchKey3":"1234","username":"user1"}}}}`,
			want:    `{"searchKey1":"########","user":{"searchKey2":{"user":{"searchKey3":"########","username":"user1"}}}}`,
		},
		{
			name:    "Masking field value with JSON Array",
			jsonStr: `[{"searchKey1":"1234","username":"user1"}]`,
			want:    `[{"searchKey1":"########","username":"user1"}]`,
		},
		{
			name:    "Masking field values, bool and numbers are not masked",
			jsonStr: `{"searchKey1":true,"searchKey2":2}`,
			want:    `{"searchKey1":true,"searchKey2":2}`,
		},
		{
			name:    "Masking primitive json",
			jsonStr: `0`,
			want:    `0`,
		},
		{
			name:    "Masking primitive json with key word",
			jsonStr: `"searchKey1"`,
			want:    `"searchKey1"`,
		},
		{
			name:    "Masking with invalid json",
			jsonStr: `"searchKey1" :`,
			want:    `"NON-JSON CONTENT"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := Mask([]byte(tt.jsonStr))
			assert.Equal(t, tt.want, string(masked))
		})
	}
}

func TestMaskingTurnedOff(t *testing.T) {
	jsonStr := `{"password":"1234","user":{"searchKey2":{"user":{"searchKey3":"1234","username":"user1"}}}}`
	t.Setenv(EnvMonacoSupportArchiveMaskKeys, "")

	masked := Mask([]byte(`{"password":"1234","user":{"searchKey2":{"user":{"searchKey3":"1234","username":"user1"}}}}`))
	assert.Equal(t, []byte(jsonStr), masked)
}
