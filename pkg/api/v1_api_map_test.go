//go:build unit
// +build unit

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

package api

import "testing"

func TestGetV2ApiId(t *testing.T) {
	type args struct {
		forV1Api Api
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Returns ID for non deprecated API",
			args{NewStandardApi("type_id", "config/v1/type", false, "", false)},
			"type_id",
		},
		{
			"Returns deprecating ID for deprecated API",
			args{NewStandardApi("type_id", "config/v1/type", false, "new_type_id", false)},
			"new_type_id",
		},
		{
			"Strips -v2 for breaking change APIs from v1",
			args{NewStandardApi("type_id-v2", "config/v1/type", true, "", false)},
			"type_id",
		},
		{
			"Strips -v2 if deprecating ID was a breaking change API in v1",
			args{NewStandardApi("og_type_id", "config/v1/type", true, "type_id-v2", false)},
			"type_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetV2ApiId(tt.args.forV1Api); got != tt.want {
				t.Errorf("GetV2ApiId() = %v, want %v", got, tt.want)
			}
		})
	}
}
