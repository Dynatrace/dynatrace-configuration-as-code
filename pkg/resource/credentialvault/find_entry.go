/*
 * @license
 * Copyright 2026 Dynatrace LLC
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

package credentialvault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dynatrace/dynatrace-configuration-as-code-core/api"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/pkg/config"
)

type lister interface {
	List(ctx context.Context) (api.PagedListResponse, error)
}

type entry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// findEntryByName looks for the specified credential entry by name.
// If there is more than one match an error is returned.
// If there is no match, empty string and nil is returned.
func findEntryByName(ctx context.Context, client lister, name string) (string, error) {
	items, err := client.List(ctx)
	if err != nil {
		return "", err
	}

	var found []entry
	for _, i := range items.All() {
		var e entry
		if err := json.Unmarshal(i, &e); err != nil {
			return "", err
		}
		if e.Name == name {
			found = append(found, e)
		}
	}

	switch {
	case len(found) == 0:
		return "", nil
	case len(found) > 1:
		var ids []string
		for _, i := range found {
			ids = append(ids, i.ID)
		}
		return "", fmt.Errorf("found more than one %s with same name (%s); matching IDs: %s", config.CredentialVaultID, name, ids)
	default:
		return found[0].ID, nil
	}
}
