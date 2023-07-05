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

package filter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFilterSlice_EmptySlice(t *testing.T) {
	sl := []int{}
	filtered := FilterSlice(sl, func(n int) bool {
		return n%2 == 0
	})
	assert.Empty(t, filtered, "Expected an empty slice when input is empty")
}

func TestFilterSlice_FilterOutEvenNumbers(t *testing.T) {
	sl := []int{1, 2, 3, 4, 5}
	filtered := FilterSlice(sl, func(n int) bool {
		return n%2 != 0
	})
	expected := []int{1, 3, 5}
	assert.Equal(t, expected, filtered, "Expected odd numbers to be filtered")
}

func TestFilterSlice_FilterOutStringsStartingWithA(t *testing.T) {
	slStrings := []string{"Apple", "Banana", "Avocado", "Orange"}
	filteredStrings := FilterSlice(slStrings, func(s string) bool {
		return s[0] != 'A'
	})
	expectedStrings := []string{"Banana", "Orange"}
	assert.Equal(t, expectedStrings, filteredStrings, "Expected strings starting with 'A' to be filtered")
}
