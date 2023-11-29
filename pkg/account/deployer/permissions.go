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

package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func FetchAvailablePermissionIDs(ctx context.Context, client *http.Client, url string) ([]string, error) {
	schema, err := fetchSchema(ctx, client, url)
	if err != nil {
		return nil, err
	}
	ids, err := parseSupportedPermissionIds(schema)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch supported permission ids: %w", err)
	}
	return ids, err
}

func fetchSchema(ctx context.Context, client *http.Client, schemaURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, schemaURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	schema, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return schema, nil
}

func parseSupportedPermissionIds(schema []byte) ([]string, error) {
	decodedSchema := make(map[string]any)
	if err := json.Unmarshal(schema, &decodedSchema); err != nil {
		return nil, err
	}

	errMsg := "failed to parse %s field"

	keys := []string{"components", "schemas", "PermissionsDto", "properties", "permissionName"}
	decodedSchemaCopy := decodedSchema

	for _, key := range keys {
		value, ok := decodedSchemaCopy[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(errMsg, key)
		}
		decodedSchemaCopy = value
	}
	enums, ok := decodedSchemaCopy["enum"].([]any)
	if !ok {
		return nil, fmt.Errorf(errMsg, "enum")
	}

	var permissionIds []string
	for _, e := range enums {
		id, ok := e.(string)
		if !ok {
			return nil, fmt.Errorf("unable to parse id")
		}
		permissionIds = append(permissionIds, id)
	}

	return permissionIds, nil
}
