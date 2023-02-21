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

package maps

import (
	"fmt"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/assert"
	"reflect"
	"testing"
)

func TestCopy(t *testing.T) {
	type args struct {
		dest     map[string]int
		source   map[string]int
		expected map[string]int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"Empty maps",
			args{
				map[string]int{},
				map[string]int{},
				map[string]int{},
			},
		},
		{
			"Empty dest",
			args{
				map[string]int{},
				map[string]int{"a": 1},
				map[string]int{"a": 1},
			},
		},
		{
			"Empty source",
			args{
				map[string]int{"a": 1},
				map[string]int{},
				map[string]int{"a": 1},
			},
		},
		{
			"simple combine",
			args{
				map[string]int{"a": 1},
				map[string]int{"b": 1},
				map[string]int{"a": 1, "b": 1},
			},
		},
		{
			"simple override",
			args{
				map[string]int{"a": 1},
				map[string]int{"a": 2},
				map[string]int{"a": 2},
			},
		},
		{
			"combined",
			args{
				map[string]int{"a": 1, "b": 2},
				map[string]int{"a": 3, "c": 5},
				map[string]int{"a": 3, "b": 2, "c": 5},
			},
		},
		{
			"nil source does not change dest",
			args{
				map[string]int{"a": 1, "b": 2},
				nil,
				map[string]int{"a": 1, "b": 2},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Copy(tt.args.dest, tt.args.source)
			assert.DeepEqual(t, tt.args.dest, tt.args.expected)
		})
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		name string
		args map[string]int
		want []string
	}{
		{
			"empty",
			map[string]int{},
			[]string{},
		},
		{
			"single",
			map[string]int{"a": 1},
			[]string{"a"},
		},
		{
			"some",
			map[string]int{"a": 1, "b": 2, "c": 3},
			[]string{"a", "b", "c"},
		},
		{
			"nil map does not error",
			nil,
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Keys(tt.args)

			assert.DeepEqual(t, got, tt.want, OrderStrings)
		})
	}
}

func TestValues(t *testing.T) {

	tests := []struct {
		name string
		args map[string]int
		want []int
	}{
		{
			"empty",
			map[string]int{},
			[]int{},
		},
		{
			"single",
			map[string]int{"a": 1},
			[]int{1},
		},
		{
			"some",
			map[string]int{"a": 1, "b": 2, "c": 3},
			[]int{1, 2, 3},
		},
		{
			"nil map does not error",
			nil,
			[]int{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Values(tt.args)

			assert.DeepEqual(t, got, tt.want, OrderInts)
		})
	}
}

func TestToStringMap(t *testing.T) {
	tests := []struct {
		name  string
		input map[interface{}]interface{}
		want  map[string]interface{}
	}{
		{
			"parses empty map",
			map[interface{}]interface{}{},
			map[string]interface{}{},
		},
		{
			"parses string map",
			map[interface{}]interface{}{
				"one": "string",
				"two": "string",
			},
			map[string]interface{}{
				"one": "string",
				"two": "string",
			},
		},
		{
			"flattens non-strings",
			map[interface{}]interface{}{
				struct {
					Value  string
					Number int
				}{
					"something",
					42,
				}: 52,
			},
			map[string]interface{}{
				fmt.Sprintf("%v", struct {
					Value  string
					Number int
				}{
					"something",
					42,
				}): 52,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToStringMap(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToStringMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

// OrderInts can be used in assert.DeepEqual to order an int-slice before comparing
var OrderInts = cmpopts.SortSlices(func(a, b int) bool {
	return a < b
})

// OrderStrings can be used in assert.DeepEqual to order a string-slice before comparing
var OrderStrings = cmpopts.SortSlices(func(a, b string) bool {
	return a < b
})
