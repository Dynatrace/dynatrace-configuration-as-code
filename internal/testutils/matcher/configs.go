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

package matcher

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config/template"
)

// ConfigsMatch checks if all configs of cs1 are present in cs2
// ignored fields: project, configId, originObjectId, environment
// Limitation: Does not work if the JSON is invalid e.g., { "myNumber": {{ .myNumber }}}
func ConfigsMatch(t *testing.T, cs1 /*subset of cs2*/ []config.Config, cs2 []config.Config) bool {
	allEqual := true
	for _, c1 := range cs1 {
		found := false
		for _, c2 := range cs2 {
			if !cmp.Equal(c1.Coordinate.Type, c2.Coordinate.Type) {
				continue
			}

			if !cmp.Equal(c1, c2, cmpopts.IgnoreFields(config.Config{}, "OriginObjectId", "Template", "Environment", "Coordinate")) {
				continue
			}

			if templateMatches(t, c1.Template, c2.Template) {
				found = true
				break
			}
		}
		t.Logf("config %v not found in %v", c1, cs2)
		allEqual = allEqual && found
	}
	return allEqual
}

// templateMatches returns true if all key-value pairs of t1 exist in t2
// Limitation: Does not work if the JSON is invalid (e.g., { "myNumber": {{ .myNumber }}}
func templateMatches(t *testing.T, t1 template.Template, t2 template.Template) bool {
	c1Content, err := t1.Content()
	require.NoError(t, err)
	c2Content, err := t2.Content()
	require.NoError(t, err)

	var c1Parsed map[string]json.RawMessage
	var c2Parsed map[string]json.RawMessage
	err = json.Unmarshal([]byte(c1Content), &c1Parsed)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(c2Content), &c2Parsed)
	require.NoError(t, err)

	return mapMatches(c1Parsed, c2Parsed)
}

// mapMatches returns true if all key-value pairs of m1 exist in m2
func mapMatches(m1 map[string]json.RawMessage, m2 map[string]json.RawMessage) bool {
	for key, val := range m1 {
		val2, ok := m2[key]
		if !ok || !cmp.Equal(val, val2) {
			return false
		}
	}
	return true
}
