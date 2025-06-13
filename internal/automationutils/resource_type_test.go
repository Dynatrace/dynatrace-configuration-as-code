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

package automationutils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/clients/automation"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

func TestClientResourceTypeFromConfigType(t *testing.T) {
	type response struct {
		resourceType automation.ResourceType
		err          error
	}
	type testCase struct {
		name     string
		given    config.AutomationResource
		expected response
	}
	testCases := []testCase{
		{
			name:  "workflow type is correctly transformed",
			given: config.Workflow,
			expected: response{
				resourceType: automation.Workflows,
				err:          nil,
			},
		},
		{
			name:  "business calendar type is correctly transformed",
			given: config.BusinessCalendar,
			expected: response{
				resourceType: automation.BusinessCalendars,
				err:          nil,
			},
		},
		{
			name:  "scheduling rules type is correctly transformed",
			given: config.SchedulingRule,
			expected: response{
				resourceType: automation.SchedulingRules,
				err:          nil,
			},
		},
		{
			name:  "invalid type causes an error",
			given: "something-else",
			expected: response{
				resourceType: -1,
				err:          fmt.Errorf("unknown automation resource type %q", "something-else"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := ClientResourceTypeFromConfigType(tc.given)
			assert.Equal(t, tc.expected.resourceType, actual)
			assert.Equal(t, tc.expected.err, err)
		})
	}
}
