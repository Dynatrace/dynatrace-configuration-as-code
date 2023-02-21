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

import "testing"

func TestIsMeId(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			"valid all uppercase no letters in the back",
			"HOST-1234123412341234",
			true,
		},
		{
			"valid back mixed back all lowercase",
			"KUBERNETES_CLUSTER-0123456789abcdef",
			true,
		},
		{
			"valid back mixed all uppercase",
			"HOST_GROUP-0123456789ABCDEF",
			true,
		},
		{
			"valid back mixed",
			"KUBERNETES_CLUSTER-0123456789AbCdEf",
			true,
		},
		{
			"to few characters in the back",
			"SERVICE-1234",
			false,
		},
		{
			"mixed in front",
			"AppLIcation-1234",
			false,
		},
		{
			"some invalid string",
			"some-string",
			false,
		},
		{
			"another invalid string",
			"some-STRING ALSO INVALID",
			false,
		},
		{
			"id encased in other stuff should not be valid",
			"some stuff KUBERNETES_CLUSTER-0123456789AbCdEf others",
			false,
		},
		{
			"id with suffix is not valid",
			"KUBERNETES_CLUSTER-0123456789AbCdEf others",
			false,
		},
		{
			"id with prefix is not valid",
			"some stuff KUBERNETES_CLUSTER-0123456789AbCdEf",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMeId(tt.id); got != tt.valid {
				t.Errorf("IsMeId() = %v, want %v", got, tt.valid)
			}
		})
	}
}
