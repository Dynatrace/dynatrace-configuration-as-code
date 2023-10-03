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

package rand

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInt(t *testing.T) {
	_, err := Int(10) // just make sure that we don't panic for every number
	assert.Nil(t, err)
}

func FuzzInt(f *testing.F) {
	seed := []int64{-1, 0, 1337, 42}
	for i := range seed {
		f.Add(seed[i])
	}

	f.Fuzz(func(t *testing.T, n int64) {
		if n <= 0 {
			return // This is not for testing
		}

		i, err := Int(n)
		assert.Nil(t, err)
		assert.True(t, i <= n)
	})
}
