/*
 * @license
 * Copyright 2025 Dynatrace LLC
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

package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	persistence "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/persistence/internal/types"
)

func TestPolicyBinding_MarshalYAML(t *testing.T) {
	type testCase struct {
		name          string
		policyBinding persistence.PolicyBinding
		expected      any
	}
	tests := []testCase{
		{
			name:          "empty",
			policyBinding: persistence.PolicyBinding{},
			expected:      "\"\"\n",
		},
		{
			name:          "policy name at root level",
			policyBinding: persistence.PolicyBinding{Value: "policy name"},
			expected: `policy name
`,
		},
		{
			name:          "policy reference at root level",
			policyBinding: persistence.PolicyBinding{Id: "policy-id", Type: "reference"},
			expected: `type: reference
id: policy-id
`,
		},
		{
			name:          "policy name nested",
			policyBinding: persistence.PolicyBinding{Policy: &persistence.Reference{Value: "policy-name"}},
			expected: `policy: policy-name
`,
		},
		{
			name:          "policy reference nested",
			policyBinding: persistence.PolicyBinding{Policy: &persistence.Reference{Id: "policy-id", Type: "reference"}},
			expected: `policy:
  type: reference
  id: policy-id
`,
		},
		{
			name: "policy reference nested with boundaries",
			policyBinding: persistence.PolicyBinding{
				Policy:     &persistence.Reference{Id: "policy-id", Type: "reference"},
				Boundaries: []persistence.Reference{{Id: "boundary-id", Type: "reference"}, {Value: "My boundary"}},
			},
			expected: `policy:
  type: reference
  id: policy-id
boundaries:
- type: reference
  id: boundary-id
- My boundary
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.policyBinding)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(got))
		})
	}

}

func TestPolicyBinding_UnmarshalYAML(t *testing.T) {
	type testCase struct {
		name     string
		yaml     string
		expected any
	}
	tests := []testCase{
		{
			name:     "empty",
			yaml:     "\"\"\n",
			expected: persistence.PolicyBinding{},
		},
		{
			name: "policy name at root level",
			yaml: `policy name
`,
			expected: persistence.PolicyBinding{Value: "policy name"},
		},
		{
			name: "policy name at root level - boundaries enabled",
			yaml: `policy name
`,
			expected: persistence.PolicyBinding{Value: "policy name"},
		},
		{
			name: "policy reference at root level",
			yaml: `type: reference
id: policy-id
`,
			expected: persistence.PolicyBinding{Id: "policy-id", Type: "reference"},
		},
		{
			name: "policy reference at root level - boundaries enabled",
			yaml: `policy name
`,
			expected: persistence.PolicyBinding{Value: "policy name"},
		},
		{
			name: "policy name nested",
			yaml: `policy: policy-name
`,
			expected: persistence.PolicyBinding{Policy: &persistence.Reference{Value: "policy-name"}},
		},
		{
			name: "policy reference nested",
			yaml: `policy:
  type: reference
  id: policy-id
`,
			expected: persistence.PolicyBinding{Policy: &persistence.Reference{Id: "policy-id", Type: "reference"}},
		},
		{
			name: "policy reference nested with boundaries",
			yaml: `policy:
  type: reference
  id: policy-id
boundaries:
- type: reference
  id: boundary-id
- My boundary
`,
			expected: persistence.PolicyBinding{
				Policy:     &persistence.Reference{Id: "policy-id", Type: "reference"},
				Boundaries: []persistence.Reference{{Id: "boundary-id", Type: "reference"}, {Value: "My boundary"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got persistence.PolicyBinding
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}

}

func TestReference_MarshalYAML(t *testing.T) {
	type testCase struct {
		name      string
		reference persistence.Reference
		expected  any
	}
	tests := []testCase{
		{
			name:      "empty",
			reference: persistence.Reference{},
			expected:  "\"\"\n",
		},
		{
			name:      "name",
			reference: persistence.Reference{Value: "policy name"},
			expected:  "policy name\n",
		},
		{
			name:      "reference",
			reference: persistence.Reference{Id: "policy-id", Type: "reference"},
			expected:  "type: reference\nid: policy-id\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := yaml.Marshal(tt.reference)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(got))
		})
	}
}

func TestReference_UnmarshalYAML(t *testing.T) {
	type testCase struct {
		name     string
		yaml     string
		expected any
	}

	tests := []testCase{
		{
			name:     "empty",
			yaml:     "",
			expected: persistence.Reference{},
		},
		{
			name:     "name",
			yaml:     "My policy",
			expected: persistence.Reference{Value: "My policy"},
		},
		{
			name:     "reference",
			yaml:     "id: policy-id\ntype: reference",
			expected: persistence.Reference{Id: "policy-id", Type: "reference"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var got persistence.Reference
			err := yaml.Unmarshal([]byte(tt.yaml), &got)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}
