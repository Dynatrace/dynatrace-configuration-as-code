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

package util

import (
	"encoding/base64"
	"fmt"
)

const format = "monaco-generated-id[%s$%s]"
const externalIdMaxLength = 500

// GenerateExternalId generates the external-id for settings 2.0 objects based on the schema, and id.
// The result of the function is pure.
// Max length for the external id is 500
func GenerateExternalId(schema, id string) string {
	fullId := fmt.Sprintf(format, schema, id)

	encodedExternalId := base64.StdEncoding.EncodeToString([]byte(fullId))

	if len(encodedExternalId) > externalIdMaxLength {
		return encodedExternalId[externalIdMaxLength:]
	}

	return encodedExternalId
}
