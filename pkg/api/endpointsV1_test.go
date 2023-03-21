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

package api

import "testing"

func TestGetV2ApiId(t *testing.T) {
	type args struct {
		forV1Api API
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Returns ID for non deprecated API",
			args{API{ID: "type_id", URLPath: "config/v1/type"}},
			"type_id",
		},
		{
			"Returns deprecating ID for deprecated API",
			args{API{ID: "type_id", URLPath: "config/v1/type", DeprecatedBy: "new_type_id"}},
			"new_type_id",
		},
		{
			"Strips -v2 for breaking change APIs from v1",
			args{API{ID: "type_id-v2", URLPath: "config/v1/type", NonUniqueName: true}},
			"type_id",
		},
		{
			"Strips -v2 if deprecating ID was a breaking change API in v1",
			args{API{ID: "og_type_id", URLPath: "config/v1/type", NonUniqueName: true, DeprecatedBy: "type_id-v2"}},
			"type_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetV2ID(tt.args.forV1Api); got != tt.want {
				t.Errorf("GetV2ID() = %v, want %v", got, tt.want)
			}
		})
	}
}
