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
package loader

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateT(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		expected       error
		expectedErrMsg string
	}{
		{

			name:           "group reference not found",
			path:           "test-resources/no-ref-group.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-group-ref": no referenced target found`,
		},
		{
			name:           "environment level policy reference not found",
			path:           "test-resources/no-ref-policy-env.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-policy-ref": no referenced target found`,
		},
		{
			name:           "account level policy reference not found",
			path:           "test-resources/no-ref-policy-account.yaml",
			expected:       ErrRefMissing,
			expectedErrMsg: `error validating account resources with id "non-existing-policy-ref": no referenced target found`,
		},
		{
			name:           "group reference with missing id field",
			path:           "test-resources/no-id-field-group-ref.yaml",
			expected:       ErrIdFieldMissing,
			expectedErrMsg: `error validating account resources: no ref id field found`,
		},
		{
			name:     "valid",
			path:     "test-resources/valid.yaml",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resources, _ := Load(afero.NewOsFs(), tc.path)
			err := Validate(resources)
			if tc.expected != nil {
				assert.ErrorIs(t, err, tc.expected)
				assert.ErrorContains(t, err, tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
