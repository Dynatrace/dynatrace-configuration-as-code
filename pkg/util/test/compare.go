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

package test

import "github.com/google/go-cmp/cmp/cmpopts"

// OrderInts can be used in assert.DeepEqual to order an int-slice before comparing
var OrderInts = cmpopts.SortSlices(func(a, b int) bool {
	return a < b
})

// OrderStrings can be used in assert.DeepEqual to order a string-slice before comparing
var OrderStrings = cmpopts.SortSlices(func(a, b string) bool {
	return a < b
})
