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

package delete

import (
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/json"
)

func GenerateJSONSchema() ([]byte, error) {
	schema, err := json.GenerateJSONSchemaString(SchemaDef{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON schema for delete file: %w", err)
	}
	return schema, nil
}
