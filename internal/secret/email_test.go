/*
 * @license
 * Copyright 2024 Dynatrace LLC
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

package secret

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		email          string
		expectedOutput string
	}{
		{"test@example.com", "te***@ex***.com"},
		{"short@ex.com", "sh***@***.com"},
		{"a@b.co.uk", "***@***.co.uk"},
		{"invalid", "***"},
		{"invalid@", "***"},
		{"invalid.com", "***"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("MaskEmail(%s)", tc.email), func(t *testing.T) {
			assert.Equal(t, tc.expectedOutput, fmt.Sprintf("%s", Email(tc.email)))
		})
	}
}
