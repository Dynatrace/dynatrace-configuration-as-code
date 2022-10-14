//go:build unit

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

package deploy

import (
	p "github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/project/v2"
	"reflect"
	"testing"
)

func Test_filterProjectsByName(t *testing.T) {
	type args struct {
		projects []p.Project
		names    []string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"returns nothing if no names given",
			args{
				[]p.Project{
					{
						Id:           "project A",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project B",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{},
			},
			nil,
			false,
		},
		{
			"filters for project by name",
			args{
				[]p.Project{
					{
						Id:           "project A",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project B",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project A"},
			},
			[]string{"project A"},
			false,
		},
		{
			"filters for grouping projects by name",
			args{
				[]p.Project{
					{
						Id:           "project.a",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project.b",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project2",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project3.a",
						GroupId:      "project3",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project"},
			},
			[]string{"project.a", "project.b"},
			false,
		},
		{
			"returns error if project of given name is not found",
			args{
				[]p.Project{
					{
						Id:           "project.a",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project.b",
						GroupId:      "project",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project2",
						GroupId:      "",
						Configs:      nil,
						Dependencies: nil,
					},
					{
						Id:           "project3.a",
						GroupId:      "project3",
						Configs:      nil,
						Dependencies: nil,
					},
				},
				[]string{"project", "UNDEFINED PROJECT"},
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterProjectsByName(tt.args.projects, tt.args.names)
			if (err != nil) != tt.wantErr {
				t.Errorf("filterProjectsByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterProjectsByName() got = %v, want %v", got, tt.want)
			}
		})
	}
}
