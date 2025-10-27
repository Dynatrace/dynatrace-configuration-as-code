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

package delete_test

import (
	"encoding/json"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	account "github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/account/delete"
)

func Test_GenerateJSONSchema(t *testing.T) {
	type testCase struct {
		name     string
		fileName string
	}

	cases := []testCase{
		{
			name:     "schema generated as expected",
			fileName: "testdata/schema.json",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectedRaw, err := afero.ReadFile(afero.NewOsFs(), tc.fileName)
			require.NoError(t, err)

			var expectedJson map[string]interface{}
			err = json.Unmarshal(expectedRaw, &expectedJson)
			require.NoError(t, err)

			gotRaw, err := account.GenerateJSONSchema()
			require.NoError(t, err)

			var gotJson map[string]interface{}
			err = json.Unmarshal(gotRaw, &gotJson)
			require.NoError(t, err)

			assert.NoError(t, err)
			assert.Equal(t, expectedJson, gotJson)
		})
	}
}
