// @license
// Copyright 2021 Dynatrace LLC
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

//go:build unit

package slices

import (
	"gotest.tools/assert"
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		value    int
		expected bool
	}{
		{
			"not contained in empty",
			[]int{},
			1,
			false,
		},
		{
			"not contained single",
			[]int{2},
			1,
			false,
		},
		{
			"not contained multiple",
			[]int{2, 3, 4},
			1,
			false,
		},
		{
			"contained single",
			[]int{1},
			1,
			true,
		},
		{
			"contained multi",
			[]int{1, 2},
			1,
			true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, Contains(test.slice, test.value), test.expected)
		})
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		a        []int
		b        []int
		expected []int
	}{
		{
			"empty sets",
			[]int{},
			[]int{},
			[]int{},
		},
		{
			"a empty",
			[]int{},
			[]int{1},
			[]int{},
		},
		{
			"b empty",
			[]int{1},
			[]int{},
			[]int{1},
		},
		{
			"same elements",
			[]int{1},
			[]int{1},
			[]int{},
		},
		{
			"a more",
			[]int{1, 2},
			[]int{2},
			[]int{1},
		},
		{
			"same more",
			[]int{1, 2},
			[]int{1, 2},
			[]int{},
		},
		{
			"a even more",
			[]int{1, 2, 3, 4, 5},
			[]int{1, 2},
			[]int{3, 4, 5},
		},
		{
			"b even more",
			[]int{1, 2, 3, 4},
			[]int{1, 2, 4},
			[]int{3},
		},
		{
			"a nil",
			nil,
			[]int{1, 2, 4},
			[]int{},
		},
		{
			"b nil",
			[]int{1, 2, 4},
			nil,
			[]int{1, 2, 4},
		},
		{
			"both nil",
			nil,
			nil,
			[]int{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.DeepEqual(t, Difference(test.a, test.b), test.expected)
		})
	}
}
