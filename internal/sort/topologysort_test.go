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

package sort

import (
	"reflect"
	"testing"
)

func TestTopologySort(t *testing.T) {
	type args struct {
		incomingEdges [][]bool
		inDegrees     []int
	}
	tests := []struct {
		name           string
		args           args
		wantTopoSorted []int
		wantErrs       []TopologySortError
	}{
		{
			"correctly sorts: 0->1->2",
			args{
				[][]bool{
					{false, false, false},
					{true, false, false},
					{false, true, false},
				},
				[]int{0, 1, 1},
			},
			[]int{0, 1, 2},
			[]TopologySortError{},
		},
		{
			"correctly sorts: 0->2->1",
			args{
				[][]bool{
					{false, false, false},
					{false, false, true},
					{true, false, false},
				},
				[]int{0, 1, 1},
			},
			[]int{0, 2, 1},
			[]TopologySortError{},
		},
		{
			"reports errors on dependency cycle 0->1->2->0",
			args{
				[][]bool{
					{false, false, true},
					{true, false, false},
					{false, true, false},
				},
				[]int{1, 1, 1},
			},
			[]int{},
			[]TopologySortError{
				{OnId: 0, UnresolvedIncomingEdgesFrom: []int{2}},
				{OnId: 1, UnresolvedIncomingEdgesFrom: []int{0}},
				{OnId: 2, UnresolvedIncomingEdgesFrom: []int{1}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTopoSorted, gotErrs := TopologySort(tt.args.incomingEdges, tt.args.inDegrees)
			if !reflect.DeepEqual(gotTopoSorted, tt.wantTopoSorted) {
				t.Errorf("TopologySort() gotTopoSorted = %v, want %v", gotTopoSorted, tt.wantTopoSorted)
			}
			if !reflect.DeepEqual(gotErrs, tt.wantErrs) {
				t.Errorf("TopologySort() gotErrs = %v, want %v", gotErrs, tt.wantErrs)
			}
		})
	}
}
