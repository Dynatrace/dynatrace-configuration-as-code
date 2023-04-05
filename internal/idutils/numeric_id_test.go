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
	"encoding/base64"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetNumericIDForObjectID(t *testing.T) {
	tests := []struct {
		name          string
		givenObjectID string
		wantNumericID int
		wantErr       bool
	}{
		{
			name:          "with new UUID #1",
			givenObjectID: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRjNDZlNDZiMy02ZDk2LTMyYTctOGI1Yi1mNjExNzcyZDAxNjW-71TeFdrerQ",
			wantNumericID: -4292415658385853785,
		},
		{
			name:          "with new UUID #2",
			givenObjectID: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACQ5ZTJhMDVlZC05OTQyLTNmOTgtODNmZS02ZTI1MWJjYzNiNTW-71TeFdrerQ",
			wantNumericID: -7049815748658446440,
		},
		{
			name:          "with legacy UUID",
			givenObjectID: "vu9U3hXa3q0AAAABABhidWlsdGluOm1hbmFnZW1lbnQtem9uZXMABnRlbmFudAAGdGVuYW50ACRkMGRlZDRhNy1mY2ZlLTQ2MDUtYTEyMy03YWE4ZDBmYTVhMja-71TeFdrerQ",
			wantNumericID: 3277109782074005416,
		},
		{
			name:          "returns error for non base64 encoded input",
			givenObjectID: "I'm not a base64 string at all",
			wantErr:       true,
		},
		{
			name:          "returns error if object ID does not contain a UUID",
			givenObjectID: base64.RawURLEncoding.EncodeToString([]byte("objectIDstuff:schema:somemoreInfo:not-a-uuid-id")),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.givenObjectID, func(t *testing.T) {
			got, err := GetNumericIDForObjectID(tt.givenObjectID)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equalf(t, tt.wantNumericID, got, "GetNumericIDForObjectID(%v):\n\twant: %064b\n\t got: %064b", tt.givenObjectID, tt.wantNumericID, got)
		})
	}
}

// this test matches the test cases of the code generating numeric IDs in Dynatrace
func TestGetLegacyNumericId(t *testing.T) {
	tests := []struct {
		name  string
		given string
		want  int
	}{
		{
			"low number",
			"0aa2a378-24c9-4967-a83c-e3de4703b9e1",
			5,
		},
		{
			"high number",
			"fcffffff-179f-4e10-b945-4ab9fea60fe9",
			3221225470,
		},
		{
			"max number",
			"feffffff-ffff-482e-9368-77e3ffffff01",
			9223372036854775807,
		},
		{
			"high negative number",
			"fbffffff-17dc-4b68-97a0-33100ce14a8e",
			-3221225470,
		},
		{
			"small negative number",
			"09c9775b-6164-40c4-b051-91370bf25b21",
			-5,
		},
		{
			"min number",
			"ffffffff-ffff-4236-bd34-0fc2ffffff01",
			-9223372036854775808,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := uuid.Parse(tt.given)
			assert.NoError(t, err)
			got, err := getLegacyNumericID(u)
			assert.NoError(t, err)
			assert.Equalf(t, tt.want, got, "getLegacyNumericID(%v)", tt.given)
		})
	}
}
